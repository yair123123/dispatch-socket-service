package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"dispatch-socket-service/internal/clients"
	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"

	"github.com/redis/go-redis/v9"
)

type DispatchRoundService struct {
	rdb                redis.UniversalClient
	offers             *OfferDeliveryService
	coreClient         clients.CoreClient
	logger             *slog.Logger
	callbackMaxRetries int
	callbackBackoff    time.Duration
}

func NewDispatchRoundService(rdb redis.UniversalClient, offers *OfferDeliveryService, coreClient clients.CoreClient, logger *slog.Logger, callbackMaxRetries int, callbackBackoff time.Duration) *DispatchRoundService {
	return &DispatchRoundService{
		rdb:                rdb,
		offers:             offers,
		coreClient:         coreClient,
		logger:             logger,
		callbackMaxRetries: callbackMaxRetries,
		callbackBackoff:    callbackBackoff,
	}
}

func (s *DispatchRoundService) StartRound(ctx context.Context, req models.StartDispatchRoundRequest) {
	candidates, err := s.findCandidates(ctx, req)
	if err != nil {
		s.logger.Error("dispatch candidate lookup failed", "round_id", req.RoundID, "error", err)
		return
	}
	if len(candidates) == 0 {
		s.reportRoundResultWithRetry(context.Background(), models.DispatchRoundResultRequest{RideID: req.RideID, RoundID: req.RoundID, RoundNumber: req.RoundNumber, Status: "no_candidates"})
		return
	}

	rideID := strconv.FormatInt(req.RideID, 10)
	expiresAt := time.Now().UTC().Add(time.Duration(req.TimeoutSeconds) * time.Second)
	_, err = s.offers.DeliverOfferBatch(ctx, models.SendOfferRequest{
		RoundID:     req.RoundID,
		RideID:      rideID,
		RoundNumber: req.RoundNumber,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		DriverIDs:   candidates,
		Payload: models.OfferDetail{
			Price: req.RidePreview.Price,
			Origin: models.LocationMeta{
				Label: req.RidePreview.OriginText,
				Lat:   req.OriginLat,
				Lon:   req.OriginLon,
			},
			Destination: models.LocationMeta{Label: req.RidePreview.DestinationText},
		},
	})
	if err != nil {
		s.logger.Error("dispatch offer delivery failed", "round_id", req.RoundID, "error", err)
		return
	}

	go s.awaitRoundTimeout(req, expiresAt)
}

func (s *DispatchRoundService) awaitRoundTimeout(req models.StartDispatchRoundRequest, expiresAt time.Time) {
	timer := time.NewTimer(time.Until(expiresAt))
	defer timer.Stop()
	<-timer.C
	rideID := strconv.FormatInt(req.RideID, 10)
	status, err := s.rdb.HGet(context.Background(), rediskeys.RideDispatchKey(rideID), "status").Result()
	if err != nil {
		s.logger.Warn("failed reading ride dispatch status at timeout", "ride_id", req.RideID, "round_id", req.RoundID, "error", err)
	}
	if status == "assigned" {
		return
	}
	s.reportRoundResultWithRetry(context.Background(), models.DispatchRoundResultRequest{RideID: req.RideID, RoundID: req.RoundID, RoundNumber: req.RoundNumber, Status: "no_accept"})
	_ = s.rdb.HSet(context.Background(), rediskeys.RideDispatchKey(rideID), "status", "closed").Err()
}

func (s *DispatchRoundService) ReportWinner(ctx context.Context, rideID int64, roundID string, roundNumber int, winnerDriverID int64) {
	s.reportRoundResultWithRetry(ctx, models.DispatchRoundResultRequest{
		RideID:         rideID,
		RoundID:        roundID,
		RoundNumber:    roundNumber,
		Status:         "winner_selected",
		WinnerDriverID: &winnerDriverID,
	})
}

func (s *DispatchRoundService) reportRoundResultWithRetry(ctx context.Context, req models.DispatchRoundResultRequest) {
	set, err := s.rdb.SetNX(ctx, rediskeys.RoundResultSentKey(req.RoundID), req.Status, 24*time.Hour).Result()
	if err != nil {
		s.logger.Error("failed to reserve round result callback", "round_id", req.RoundID, "status", req.Status, "error", err)
		return
	}
	if !set {
		return
	}

	var lastErr error
	for attempt := 1; attempt <= s.callbackMaxRetries; attempt++ {
		if err := s.coreClient.ReportDispatchRoundResult(ctx, req); err == nil {
			s.logger.Info("dispatch round result reported", "round_id", req.RoundID, "status", req.Status, "attempt", attempt)
			return
		} else {
			lastErr = err
			time.Sleep(time.Duration(attempt) * s.callbackBackoff)
		}
	}
	s.logger.Error("dispatch round result callback failed", "round_id", req.RoundID, "status", req.Status, "error", lastErr)
}

func (s *DispatchRoundService) findCandidates(ctx context.Context, req models.StartDispatchRoundRequest) ([]string, error) {
	results, err := s.rdb.GeoRadius(ctx, rediskeys.DriversLocationsKey, req.OriginLon, req.OriginLat, &redis.GeoRadiusQuery{
		Radius: req.RadiusKm,
		Unit:   "km",
		Sort:   "ASC",
		Count:  req.MaxCandidates * 3,
	}).Result()
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0, req.MaxCandidates)
	for _, result := range results {
		if len(candidates) >= req.MaxCandidates {
			break
		}
		driverID := result.Name
		state, err := s.rdb.HGetAll(ctx, rediskeys.DriverStateKey(driverID)).Result()
		if err != nil {
			return nil, err
		}
		if state["is_online"] != "true" || state["is_available"] != "true" {
			continue
		}
		candidates = append(candidates, driverID)
	}
	return candidates, nil
}

func ParseInt64(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse int64 %q: %w", s, err)
	}
	return v, nil
}
