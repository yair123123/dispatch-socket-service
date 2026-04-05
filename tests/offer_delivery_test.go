package tests

import (
	"context"
	"testing"
	"time"

	"dispatch-socket-service/internal/models"
	"dispatch-socket-service/internal/services"
	"dispatch-socket-service/internal/ws"

	"github.com/stretchr/testify/require"
)

func TestOfferDeliveryConnectedVsNotConnected(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	cm := ws.NewConnectionManager()
	server, client, cleanup := makeWebsocketPair(t)
	defer cleanup()
	cm.Set("driver_1", server)

	svc := services.NewOfferDeliveryService(rdb, cm, time.Second)
	resp, err := svc.DeliverOfferBatch(context.Background(), models.SendOfferRequest{
		RideID: "ride_1", RoundNumber: 1, ExpiresAt: time.Now().Add(time.Minute).UTC().Format(time.RFC3339),
		DriverIDs: []string{"driver_1", "driver_2"},
		Payload:   models.OfferDetail{Price: 10},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"driver_1"}, resp.SentDriverIDs)
	require.Equal(t, []string{"driver_2"}, resp.NotConnectedDriverIDs)

	_, msg, err := client.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(msg), "ride_offer_created")
}
