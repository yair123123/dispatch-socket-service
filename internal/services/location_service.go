package services

import (
	"context"
	"fmt"
	"time"

	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/utils"

	"github.com/go-redis/redis/v9"
)

type LocationService struct {
	rdb      redis.UniversalClient
	stateTTL time.Duration
	geoTTL   time.Duration
}

func NewLocationService(rdb redis.UniversalClient, stateTTL, geoTTL time.Duration) *LocationService {
	return &LocationService{rdb: rdb, stateTTL: stateTTL, geoTTL: geoTTL}
}

func (s *LocationService) UpdateLocation(ctx context.Context, driverID string, p models.LocationUpdateRequest) error {
	if err := s.rdb.GeoAdd(ctx, rediskeys.DriversLocationsKey, &redis.GeoLocation{
		Name:      driverID,
		Longitude: p.Lon,
		Latitude:  p.Lat,
	}).Err(); err != nil {
		return err
	}
	key := rediskeys.DriverStateKey(driverID)
	values := map[string]interface{}{
		"driver_id":    driverID,
		"lat":          fmt.Sprintf("%f", p.Lat),
		"lon":          fmt.Sprintf("%f", p.Lon),
		"accuracy":     fmt.Sprintf("%f", p.Accuracy),
		"speed":        fmt.Sprintf("%f", p.Speed),
		"heading":      fmt.Sprintf("%f", p.Heading),
		"last_seen_at": utils.NowUTC().Format(time.RFC3339),
	}
	if err := s.rdb.HSet(ctx, key, values).Err(); err != nil {
		return err
	}
	if err := s.rdb.Expire(ctx, key, s.stateTTL).Err(); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, rediskeys.DriversLocationsKey, s.geoTTL).Err()
}
