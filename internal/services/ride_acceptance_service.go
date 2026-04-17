package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"

	"github.com/redis/go-redis/v9"
)

type RideAcceptanceService struct {
	rdb          redis.UniversalClient
	connections  DriverSender
	offers       *OfferDeliveryService
	coreSync     *CoreSyncService
	dispatch     *DispatchRoundService
	writeTimeout time.Duration
	logger       *slog.Logger
}

type AcceptResult struct {
	Success bool
	Reason  string
}

var acceptLua = redis.NewScript(`
local dispatchKey = KEYS[1]
local offerKey = KEYS[2]
local winnerKey = KEYS[3]

local driverId = ARGV[1]
local offerId = ARGV[2]
local round = tonumber(ARGV[3])
local nowEpoch = tonumber(ARGV[4])

if redis.call('EXISTS', dispatchKey) == 0 then return {'0', 'ride_not_found'} end
local status = redis.call('HGET', dispatchKey, 'status')
if status ~= 'open' then return {'0', 'ride_not_open'} end
local currentRound = tonumber(redis.call('HGET', dispatchKey, 'current_round'))
if currentRound ~= round then return {'0', 'wrong_round'} end
if redis.call('EXISTS', offerKey) == 0 then return {'0', 'offer_not_found'} end
local offerStatus = redis.call('HGET', offerKey, 'status')
if offerStatus ~= 'pending' then return {'0', 'offer_not_pending'} end
local keyOfferId = redis.call('HGET', offerKey, 'offer_id')
if keyOfferId ~= offerId then return {'0', 'offer_mismatch'} end
local expiresAtEpoch = tonumber(redis.call('HGET', offerKey, 'expires_at_epoch'))
if expiresAtEpoch ~= nil and expiresAtEpoch < nowEpoch then return {'0', 'offer_expired'} end
if redis.call('EXISTS', winnerKey) == 1 then return {'0', 'already_taken'} end
redis.call('SET', winnerKey, driverId)
redis.call('HSET', dispatchKey, 'status', 'assigned', 'winner_driver_id', driverId)
redis.call('HSET', offerKey, 'status', 'accepted', 'accepted_at_epoch', tostring(nowEpoch))
return {'1', 'won'}
`)

func NewRideAcceptanceService(rdb redis.UniversalClient, cm DriverSender, offers *OfferDeliveryService, coreSync *CoreSyncService, dispatch *DispatchRoundService, writeTimeout time.Duration, logger *slog.Logger) *RideAcceptanceService {
	return &RideAcceptanceService{rdb: rdb, connections: cm, offers: offers, coreSync: coreSync, dispatch: dispatch, writeTimeout: writeTimeout, logger: logger}
}

func (s *RideAcceptanceService) AcceptRide(ctx context.Context, driverID string, req models.AcceptRideRequest) (AcceptResult, error) {
	now := time.Now().Unix()
	keys := []string{rediskeys.RideDispatchKey(req.RideID), rediskeys.RideOfferKey(req.RideID, driverID), rediskeys.RideWinnerKey(req.RideID)}
	res, err := acceptLua.Run(ctx, s.rdb, keys, driverID, req.OfferID, req.RoundNumber, now).Result()
	if err != nil {
		return AcceptResult{}, err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return AcceptResult{}, fmt.Errorf("unexpected lua response: %v", res)
	}
	success := fmt.Sprint(arr[0]) == "1"
	reason := fmt.Sprint(arr[1])
	if !success {
		rej := models.Envelope{Type: "ride_accept_rejected", Payload: models.RideAcceptRejectedPayload{RideID: req.RideID, OfferID: req.OfferID, Reason: reason}}
		_ = s.connections.SendToDriver(driverID, rej, s.writeTimeout)
		return AcceptResult{Success: false, Reason: reason}, nil
	}
	offerRaw, err := s.rdb.HGetAll(ctx, rediskeys.RideOfferKey(req.RideID, driverID)).Result()
	if err != nil {
		return AcceptResult{}, err
	}
	price, _ := strconv.ParseFloat(offerRaw["price"], 64)
	assigned := models.Envelope{Type: "ride_assigned", Payload: models.RideAssignedPayload{
		RideOfferPayload: models.RideOfferPayload{RideID: req.RideID, OfferID: req.OfferID, RoundNumber: req.RoundNumber, Price: price},
		NextStep:         "go_to_pickup",
	}}
	_ = s.connections.SendToDriver(driverID, assigned, s.writeTimeout)
	if err := s.offers.CancelOthers(ctx, req.RideID, req.OfferID, req.RoundNumber, driverID); err != nil {
		s.logger.Error("cancel others failed", "ride_id", req.RideID, "error", err)
	}
	dispatchMeta, metaErr := s.rdb.HMGet(ctx, rediskeys.RideDispatchKey(req.RideID), "round_id", "current_round").Result()
	if metaErr == nil && len(dispatchMeta) == 2 && dispatchMeta[0] != nil {
		roundID := fmt.Sprint(dispatchMeta[0])
		if rideIDInt, err := ParseInt64(req.RideID); err == nil {
			if winnerIDInt, err := ParseInt64(driverID); err == nil && s.dispatch != nil {
				s.dispatch.ReportWinner(context.Background(), rideIDInt, roundID, req.RoundNumber, winnerIDInt)
			}
		}
	}
	if s.coreSync != nil {
		go s.coreSync.SyncAssignment(context.Background(), models.CoreAssignDriverRequest{RideID: req.RideID, DriverID: driverID, OfferID: req.OfferID, RoundNumber: req.RoundNumber})
	}
	return AcceptResult{Success: true, Reason: "won"}, nil
}

func (s *RideAcceptanceService) RejectRide(ctx context.Context, driverID string, req models.RejectRideRequest) error {
	return s.rdb.HSet(ctx, rediskeys.RideOfferKey(req.RideID, driverID), "status", "rejected").Err()
}
