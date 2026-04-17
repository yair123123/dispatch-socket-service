package tests

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"dispatch-socket-service/internal/geo"
	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/services"
	"dispatch-socket-service/internal/ws"

	"github.com/stretchr/testify/require"
)

func TestLocationIndexesDriverIntoCellSet(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	h3 := geo.NewH3Indexer()
	location := services.NewLocationService(rdb, time.Minute, time.Minute, 9, h3)
	require.NoError(t, location.UpdateLocation(context.Background(), "10", models.LocationUpdateRequest{Lat: 31.778, Lon: 35.235}))

	state, err := rdb.HGetAll(context.Background(), rediskeys.DriverStateKey("10")).Result()
	require.NoError(t, err)
	cell := state["h3_cell"]
	require.NotEmpty(t, cell)
	members, err := rdb.SMembers(context.Background(), rediskeys.H3CellDriversKey(cell)).Result()
	require.NoError(t, err)
	require.Contains(t, members, "10")
}

func TestRingSizeZeroMatchesOnlyOriginCell(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	h3 := geo.NewH3Indexer()
	location := services.NewLocationService(rdb, time.Minute, time.Minute, 9, h3)
	offers := services.NewOfferDeliveryService(rdb, ws.NewConnectionManager(), time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond, 9, h3)

	require.NoError(t, location.UpdateLocation(context.Background(), "11", models.LocationUpdateRequest{Lat: 31.778, Lon: 35.235}))
	require.NoError(t, location.UpdateLocation(context.Background(), "12", models.LocationUpdateRequest{Lat: 31.7785, Lon: 35.2365}))
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("11"), map[string]any{"is_online": "true", "is_available": "true"}).Err())
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("12"), map[string]any{"is_online": "true", "is_available": "true"}).Err())

	rounds.StartRound(context.Background(), models.StartDispatchRoundRequest{
		RideID: 901, RoundID: "r901", RoundNumber: 1,
		OriginLat: 31.778, OriginLon: 35.235,
		TimeoutSeconds: 5, MaxCandidates: 10,
		H3Resolution: 9, RingSize: 0,
		RidePreview: models.DispatchPreview{Price: 10},
	})

	drivers, err := rdb.SMembers(context.Background(), rediskeys.RideRoundDriversKey("901", 1)).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"11"}, drivers)
}

func TestRingExpansionIncludesOuterCells(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	h3 := geo.NewH3Indexer()
	location := services.NewLocationService(rdb, time.Minute, time.Minute, 9, h3)
	offers := services.NewOfferDeliveryService(rdb, ws.NewConnectionManager(), time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond, 9, h3)

	require.NoError(t, location.UpdateLocation(context.Background(), "21", models.LocationUpdateRequest{Lat: 31.778, Lon: 35.235}))
	require.NoError(t, location.UpdateLocation(context.Background(), "22", models.LocationUpdateRequest{Lat: 31.7782, Lon: 35.2358}))
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("21"), map[string]any{"is_online": "true", "is_available": "true"}).Err())
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("22"), map[string]any{"is_online": "true", "is_available": "true"}).Err())

	rounds.StartRound(context.Background(), models.StartDispatchRoundRequest{
		RideID: 902, RoundID: "r902", RoundNumber: 1,
		OriginLat: 31.778, OriginLon: 35.235,
		TimeoutSeconds: 5, MaxCandidates: 10,
		H3Resolution: 9, RingSize: 2,
		RidePreview: models.DispatchPreview{Price: 10},
	})

	drivers, err := rdb.SMembers(context.Background(), rediskeys.RideRoundDriversKey("902", 1)).Result()
	require.NoError(t, err)
	require.Len(t, drivers, 2)
	require.Contains(t, drivers, "21")
	require.Contains(t, drivers, "22")
}

func TestCandidateFilterAndMaxCandidates(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	h3 := geo.NewH3Indexer()
	location := services.NewLocationService(rdb, time.Minute, time.Minute, 9, h3)
	offers := services.NewOfferDeliveryService(rdb, ws.NewConnectionManager(), time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond, 9, h3)

	for _, id := range []string{"31", "32", "33"} {
		require.NoError(t, location.UpdateLocation(context.Background(), id, models.LocationUpdateRequest{Lat: 31.778, Lon: 35.235}))
	}
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("31"), map[string]any{"is_online": "true", "is_available": "true"}).Err())
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("32"), map[string]any{"is_online": "false", "is_available": "true"}).Err())
	require.NoError(t, rdb.HSet(context.Background(), rediskeys.DriverStateKey("33"), map[string]any{"is_online": "true", "is_available": "true"}).Err())

	rounds.StartRound(context.Background(), models.StartDispatchRoundRequest{
		RideID: 903, RoundID: "r903", RoundNumber: 1,
		OriginLat: 31.778, OriginLon: 35.235,
		TimeoutSeconds: 5, MaxCandidates: 1,
		H3Resolution: 9, RingSize: 0,
		RidePreview: models.DispatchPreview{Price: 10},
	})
	drivers, err := rdb.SMembers(context.Background(), rediskeys.RideRoundDriversKey("903", 1)).Result()
	require.NoError(t, err)
	require.Len(t, drivers, 1)
	require.Equal(t, "31", drivers[0])
}

func TestResolutionMismatchSkipsRoundMatching(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{}
	h3 := geo.NewH3Indexer()
	offers := services.NewOfferDeliveryService(rdb, ws.NewConnectionManager(), time.Second)
	rounds := services.NewDispatchRoundService(rdb, offers, core, slog.Default(), 2, 5*time.Millisecond, 9, h3)

	rounds.StartRound(context.Background(), models.StartDispatchRoundRequest{
		RideID: 904, RoundID: "r904", RoundNumber: 2,
		OriginLat: 31.778, OriginLon: 35.235,
		TimeoutSeconds: 5, MaxCandidates: 5,
		H3Resolution: 8, RingSize: 2,
		RidePreview: models.DispatchPreview{Price: 11},
	})

	require.Len(t, core.roundResults, 1)
	require.Equal(t, "no_candidates", core.roundResults[0].Status)
}
