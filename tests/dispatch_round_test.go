package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpserver "dispatch-socket-service/internal/http"
	"dispatch-socket-service/internal/http/handlers"
	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/services"
	"dispatch-socket-service/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestStartRoundNoCandidatesReportsNoCandidates(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	offers := services.NewOfferDeliveryService(rdb, ws.NewConnectionManager(), time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond)

	rounds.StartRound(context.Background(), models.StartDispatchRoundRequest{
		RideID: 101, RoundID: "ride_101_round_1", RoundNumber: 1,
		OriginLat: 31.7, OriginLon: 35.2, RadiusKm: 2, TimeoutSeconds: 1, MaxCandidates: 5,
		RidePreview: models.DispatchPreview{OriginText: "A", DestinationText: "B", Price: 10},
	})

	require.Len(t, core.roundResults, 1)
	require.Equal(t, "no_candidates", core.roundResults[0].Status)
}

func TestStartRoundTimeoutReportsNoAccept(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	offers := services.NewOfferDeliveryService(rdb, ws.NewConnectionManager(), time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond)

	require.NoError(t, rdb.GeoAdd(context.Background(), rediskeys.DriversLocationsKey, &redis.GeoLocation{Name: "7", Longitude: 35.235, Latitude: 31.778}).Err())
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("7"), map[string]any{"is_online": "true", "is_available": "true"}).Err())

	rounds.StartRound(context.Background(), models.StartDispatchRoundRequest{
		RideID: 202, RoundID: "ride_202_round_1", RoundNumber: 1,
		OriginLat: 31.778, OriginLon: 35.235, RadiusKm: 1, TimeoutSeconds: 1, MaxCandidates: 5,
		RidePreview: models.DispatchPreview{OriginText: "A", DestinationText: "B", Price: 10},
	})

	time.Sleep(1200 * time.Millisecond)
	require.Len(t, core.roundResults, 1)
	require.Equal(t, "no_accept", core.roundResults[0].Status)
}

func TestInvalidInternalSecretRejected(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	cm := ws.NewConnectionManager()
	offers := services.NewOfferDeliveryService(rdb, cm, time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond)
	handler := handlers.NewInternalDispatchHandler(offers, rounds)
	router := gin.New()
	internal := router.Group("/internal")
	internal.Use(httpserver.InternalAuthMiddleware("secret"))
	internal.POST("/dispatch/start-round", handler.StartRound)

	body, _ := json.Marshal(models.StartDispatchRoundRequest{RideID: 1, RoundID: "r1", RoundNumber: 1, OriginLat: 1, OriginLon: 2, RadiusKm: 1, TimeoutSeconds: 1, MaxCandidates: 1, RidePreview: models.DispatchPreview{}})
	req := httptest.NewRequest(http.MethodPost, "/internal/dispatch/start-round", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
