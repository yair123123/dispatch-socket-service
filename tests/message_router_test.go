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

	"github.com/stretchr/testify/require"
)

func TestWebSocketMessageRoutingBasic(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	presence := services.NewPresenceService(rdb, time.Minute)
	location := services.NewLocationService(rdb, time.Minute, time.Minute)
	cm := ws.NewConnectionManager()
	core := &fakeCoreClient{}
	retry := services.NewRetrySyncService(rdb, core, slog.Default(), time.Second, 2)
	sync := services.NewCoreSyncService(core, retry, slog.Default())
	offers := services.NewOfferDeliveryService(rdb, cm, time.Second)
	accept := services.NewRideAcceptanceService(rdb, cm, offers, sync, time.Second, slog.Default())
	router := ws.NewMessageRouter(presence, location, accept, slog.Default())

	err := router.Route(context.Background(), "driver_1", []byte(`{"type":"set_availability","payload":{"is_available":true}}`))
	require.NoError(t, err)
	state, err := rdb.HGetAll(context.Background(), rediskeys.DriverStateKey("driver_1")).Result()
	require.NoError(t, err)
	require.Equal(t, "true", state["is_available"])

	err = router.Route(context.Background(), "driver_1", []byte(`{"type":"location_update","payload":{"lat":1,"lon":2,"accuracy":1,"speed":1,"heading":1,"timestamp":"2026-04-05T18:30:00Z"}}`))
	require.NoError(t, err)

	_, err = offers.DeliverOfferBatch(context.Background(), models.SendOfferRequest{RideID: "r1", RoundNumber: 1, ExpiresAt: time.Now().Add(time.Minute).UTC().Format(time.RFC3339), DriverIDs: []string{"driver_1"}, Payload: models.OfferDetail{Price: 1}})
	require.NoError(t, err)
	oid, err := rdb.HGet(context.Background(), rediskeys.RideOfferKey("r1", "driver_1"), "offer_id").Result()
	require.NoError(t, err)
	err = router.Route(context.Background(), "driver_1", []byte(`{"type":"reject_ride","payload":{"ride_id":"r1","offer_id":"`+oid+`","round_number":1}}`))
	require.NoError(t, err)
}
