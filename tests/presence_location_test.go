package tests

import (
	"context"
	"testing"
	"time"

	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/services"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func setupRedis(t *testing.T) (*redis.Client, func()) {
	m := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: m.Addr()})
	return c, func() { _ = c.Close(); m.Close() }
}

func TestLocationUpdateWritesRedis(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	svc := services.NewLocationService(rdb, time.Minute, time.Minute)
	err := svc.UpdateLocation(context.Background(), "driver_1", models.LocationUpdateRequest{Lat: 1.1, Lon: 2.2, Accuracy: 3.3, Speed: 4.4, Heading: 5.5})
	require.NoError(t, err)

	state, err := rdb.HGetAll(context.Background(), rediskeys.DriverStateKey("driver_1")).Result()
	require.NoError(t, err)
	require.Equal(t, "1.100000", state["lat"])
	require.Equal(t, "2.200000", state["lon"])
}

func TestAvailabilityUpdateWritesRedis(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	svc := services.NewPresenceService(rdb, time.Minute)
	err := svc.SetAvailability(context.Background(), "driver_1", true)
	require.NoError(t, err)
	state, err := rdb.HGetAll(context.Background(), rediskeys.DriverStateKey("driver_1")).Result()
	require.NoError(t, err)
	require.Equal(t, "true", state["is_available"])
}
