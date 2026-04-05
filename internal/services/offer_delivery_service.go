package services

import (
	"context"
	"time"

	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type OfferDeliveryService struct {
	rdb          redis.UniversalClient
	connections  DriverSender
	writeTimeout time.Duration
}

func NewOfferDeliveryService(rdb redis.UniversalClient, cm DriverSender, writeTimeout time.Duration) *OfferDeliveryService {
	return &OfferDeliveryService{rdb: rdb, connections: cm, writeTimeout: writeTimeout}
}

func (s *OfferDeliveryService) DeliverOfferBatch(ctx context.Context, req models.SendOfferRequest) (models.SendOfferResponse, error) {
	offerID := uuid.NewString()
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return models.SendOfferResponse{}, err
	}
	dispatchKey := rediskeys.RideDispatchKey(req.RideID)
	if err := s.rdb.HSet(ctx, dispatchKey, map[string]interface{}{
		"ride_id":          req.RideID,
		"status":           "open",
		"current_round":    req.RoundNumber,
		"expires_at":       req.ExpiresAt,
		"expires_at_epoch": expiresAt.Unix(),
		"winner_driver_id": "",
	}).Err(); err != nil {
		return models.SendOfferResponse{}, err
	}
	driversSetKey := rediskeys.RideRoundDriversKey(req.RideID, req.RoundNumber)
	if len(req.DriverIDs) > 0 {
		members := make([]interface{}, 0, len(req.DriverIDs))
		for _, d := range req.DriverIDs {
			members = append(members, d)
			if err := s.rdb.HSet(ctx, rediskeys.RideOfferKey(req.RideID, d), map[string]interface{}{
				"ride_id":          req.RideID,
				"driver_id":        d,
				"offer_id":         offerID,
				"round_number":     req.RoundNumber,
				"status":           "pending",
				"expires_at":       req.ExpiresAt,
				"expires_at_epoch": expiresAt.Unix(),
				"price":            req.Payload.Price,
				"sent_at":          time.Now().UTC().Format(time.RFC3339),
			}).Err(); err != nil {
				return models.SendOfferResponse{}, err
			}
		}
		if err := s.rdb.SAdd(ctx, driversSetKey, members...).Err(); err != nil {
			return models.SendOfferResponse{}, err
		}
	}

	event := models.Envelope{Type: "ride_offer_created", Payload: models.RideOfferPayload{
		RideID: req.RideID, OfferID: offerID, RoundNumber: req.RoundNumber,
		ExpiresAt: req.ExpiresAt, Price: req.Payload.Price,
		Origin: req.Payload.Origin, Destination: req.Payload.Destination,
	}}
	sent, notConnected, failed := s.connections.SendToDrivers(req.DriverIDs, event, s.writeTimeout)
	return models.SendOfferResponse{
		Success: len(failed) == 0,
		RideID:  req.RideID, RoundNumber: req.RoundNumber,
		SentDriverIDs: sent, NotConnectedDriverIDs: notConnected, FailedDriverIDs: failed,
	}, nil
}

func (s *OfferDeliveryService) CancelOffer(ctx context.Context, req models.CancelOfferRequest) error {
	for _, driverID := range req.DriverIDs {
		_ = s.rdb.HSet(ctx, rediskeys.RideOfferKey(req.RideID, driverID), "status", "canceled").Err()
	}
	event := models.Envelope{Type: "ride_offer_canceled", Payload: models.RideOfferCanceledPayload{RideID: req.RideID, OfferID: req.OfferID, Reason: req.Reason}}
	_, _, _ = s.connections.SendToDrivers(req.DriverIDs, event, s.writeTimeout)
	if req.RoundNumber > 0 {
		_ = s.rdb.HSet(ctx, rediskeys.RideDispatchKey(req.RideID), "status", "closed").Err()
	}
	return nil
}

func (s *OfferDeliveryService) CancelOthers(ctx context.Context, rideID, offerID string, round int, winnerDriverID string) error {
	driverIDs, err := s.rdb.SMembers(ctx, rediskeys.RideRoundDriversKey(rideID, round)).Result()
	if err != nil {
		return err
	}
	others := make([]string, 0, len(driverIDs))
	for _, d := range driverIDs {
		if d == winnerDriverID {
			continue
		}
		others = append(others, d)
		_ = s.rdb.HSet(ctx, rediskeys.RideOfferKey(rideID, d), "status", "canceled").Err()
	}
	event := models.Envelope{Type: "ride_offer_canceled", Payload: models.RideOfferCanceledPayload{RideID: rideID, OfferID: offerID, Reason: "taken_by_other_driver"}}
	_, _, _ = s.connections.SendToDrivers(others, event, s.writeTimeout)
	return nil
}
