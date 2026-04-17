package tests

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/services"
	"dispatch-socket-service/internal/ws"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type fakeCoreClient struct {
	calls        int
	fail         bool
	roundResults []models.DispatchRoundResultRequest
}

func (f *fakeCoreClient) AssignDriver(ctx context.Context, req models.CoreAssignDriverRequest) error {
	f.calls++
	if f.fail {
		return context.DeadlineExceeded
	}
	return nil
}

func (f *fakeCoreClient) ReportDispatchRoundResult(ctx context.Context, req models.DispatchRoundResultRequest) error {
	f.roundResults = append(f.roundResults, req)
	return nil
}

func buildAcceptance(t *testing.T) (*redis.Client, *services.RideAcceptanceService, *services.OfferDeliveryService, func()) {
	rdb, done := setupRedis(t)
	cm := ws.NewConnectionManager()
	logger := slog.Default()
	core := &fakeCoreClient{}
	retry := services.NewRetrySyncService(rdb, core, logger, 10*time.Millisecond, 2)
	sync := services.NewCoreSyncService(core, retry, logger)
	offers := services.NewOfferDeliveryService(rdb, cm, time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, logger, 2, 10*time.Millisecond)
	accept := services.NewRideAcceptanceService(rdb, cm, offers, sync, rounds, time.Second, logger)
	return rdb, accept, offers, done
}

func mustOfferID(t *testing.T, rdb *redis.Client, rideID, driverID string) string {
	t.Helper()
	id, err := rdb.HGet(context.Background(), rediskeys.RideOfferKey(rideID, driverID), "offer_id").Result()
	require.NoError(t, err)
	return id
}

func TestAtomicAcceptWinnerAndSecondRejected(t *testing.T) {
	rdb, accept, offers, done := buildAcceptance(t)
	defer done()
	_, err := offers.DeliverOfferBatch(context.Background(), models.SendOfferRequest{RoundID: "ride_1_round_1", RideID: "ride_1", RoundNumber: 1, ExpiresAt: time.Now().Add(time.Minute).UTC().Format(time.RFC3339), DriverIDs: []string{"d1", "d2"}, Payload: models.OfferDetail{Price: 1}})
	require.NoError(t, err)

	r1, err := accept.AcceptRide(context.Background(), "d1", models.AcceptRideRequest{RideID: "ride_1", OfferID: mustOfferID(t, rdb, "ride_1", "d1"), RoundNumber: 1})
	require.NoError(t, err)
	require.True(t, r1.Success)
	r2, err := accept.AcceptRide(context.Background(), "d2", models.AcceptRideRequest{RideID: "ride_1", OfferID: mustOfferID(t, rdb, "ride_1", "d2"), RoundNumber: 1})
	require.NoError(t, err)
	require.False(t, r2.Success)
	require.Equal(t, "ride_not_open", r2.Reason)
}

func TestMultipleAcceptsProduceSingleWinnerCallback(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	cm := ws.NewConnectionManager()
	logger := slog.Default()
	core := &fakeCoreClient{}
	retry := services.NewRetrySyncService(rdb, core, logger, 10*time.Millisecond, 2)
	sync := services.NewCoreSyncService(core, retry, logger)
	offers := services.NewOfferDeliveryService(rdb, cm, time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, logger, 2, 10*time.Millisecond)
	accept := services.NewRideAcceptanceService(rdb, cm, offers, sync, rounds, time.Second, logger)
	_, err := offers.DeliverOfferBatch(context.Background(), models.SendOfferRequest{
		RoundID: "ride_9_round_1", RideID: "9", RoundNumber: 1, ExpiresAt: time.Now().Add(time.Minute).UTC().Format(time.RFC3339),
		DriverIDs: []string{"11", "12"}, Payload: models.OfferDetail{Price: 1},
	})
	require.NoError(t, err)
	_, _ = accept.AcceptRide(context.Background(), "11", models.AcceptRideRequest{RideID: "9", OfferID: mustOfferID(t, rdb, "9", "11"), RoundNumber: 1})
	_, _ = accept.AcceptRide(context.Background(), "12", models.AcceptRideRequest{RideID: "9", OfferID: mustOfferID(t, rdb, "9", "12"), RoundNumber: 1})
	require.Len(t, core.roundResults, 1)
	require.Equal(t, "winner_selected", core.roundResults[0].Status)
	require.NotNil(t, core.roundResults[0].WinnerDriverID)
}

func TestAtomicAcceptExpiredOfferRejected(t *testing.T) {
	rdb, accept, offers, done := buildAcceptance(t)
	defer done()
	_, err := offers.DeliverOfferBatch(context.Background(), models.SendOfferRequest{RideID: "ride_2", RoundNumber: 1, ExpiresAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), DriverIDs: []string{"d1"}, Payload: models.OfferDetail{Price: 1}})
	require.NoError(t, err)
	r, err := accept.AcceptRide(context.Background(), "d1", models.AcceptRideRequest{RideID: "ride_2", OfferID: mustOfferID(t, rdb, "ride_2", "d1"), RoundNumber: 1})
	require.NoError(t, err)
	require.False(t, r.Success)
	require.Equal(t, "offer_expired", r.Reason)
}

func TestAtomicAcceptWrongRoundRejected(t *testing.T) {
	rdb, accept, offers, done := buildAcceptance(t)
	defer done()
	_, err := offers.DeliverOfferBatch(context.Background(), models.SendOfferRequest{RideID: "ride_3", RoundNumber: 2, ExpiresAt: time.Now().Add(time.Minute).UTC().Format(time.RFC3339), DriverIDs: []string{"d1"}, Payload: models.OfferDetail{Price: 1}})
	require.NoError(t, err)
	r, err := accept.AcceptRide(context.Background(), "d1", models.AcceptRideRequest{RideID: "ride_3", OfferID: mustOfferID(t, rdb, "ride_3", "d1"), RoundNumber: 1})
	require.NoError(t, err)
	require.False(t, r.Success)
	require.Equal(t, "wrong_round", r.Reason)
}
