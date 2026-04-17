package services

import (
	"context"
	"fmt"
	"time"

	"dispatch-socket-service/internal/geo"
	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/utils"

	"github.com/redis/go-redis/v9"
)

type LocationService struct {
	rdb          redis.UniversalClient
	stateTTL     time.Duration
	geoTTL       time.Duration
	h3Resolution int
	h3Indexer    *geo.H3Indexer
}

func NewLocationService(rdb redis.UniversalClient, stateTTL, geoTTL time.Duration, h3Resolution int, h3Indexer *geo.H3Indexer) *LocationService {
	return &LocationService{rdb: rdb, stateTTL: stateTTL, geoTTL: geoTTL, h3Resolution: h3Resolution, h3Indexer: h3Indexer}
}

func (s *LocationService) UpdateLocation(ctx context.Context, driverID string, p models.LocationUpdateRequest) error {
	if err := s.rdb.GeoAdd(ctx, rediskeys.DriversLocationsKey, &redis.GeoLocation{
		Name:      driverID,
		Longitude: p.Lon,
		Latitude:  p.Lat,
	}).Err(); err != nil {
		return err
	}
	cellID, err := s.h3Indexer.CellFromLatLon(p.Lat, p.Lon, s.h3Resolution)
	if err != nil {
		return err
	}
	if err := s.updateDriverH3Cell(ctx, driverID, cellID); err != nil {
		return err
	}
	key := rediskeys.DriverStateKey(driverID)
	values := map[string]interface{}{
		"driver_id":     driverID,
		"lat":           fmt.Sprintf("%f", p.Lat),
		"lon":           fmt.Sprintf("%f", p.Lon),
		"accuracy":      fmt.Sprintf("%f", p.Accuracy),
		"speed":         fmt.Sprintf("%f", p.Speed),
		"heading":       fmt.Sprintf("%f", p.Heading),
		"h3_cell":       cellID,
		"h3_resolution": fmt.Sprintf("%d", s.h3Resolution),
		"last_seen_at":  utils.NowUTC().Format(time.RFC3339),
	}
	if err := s.rdb.HSet(ctx, key, values).Err(); err != nil {
		return err
	}
	if err := s.rdb.Expire(ctx, key, s.stateTTL).Err(); err != nil {
		return err
	}
	if err := s.rdb.Expire(ctx, rediskeys.DriversLocationsKey, s.geoTTL).Err(); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, rediskeys.H3CellDriversKey(cellID), s.geoTTL).Err()
}

func (s *LocationService) updateDriverH3Cell(ctx context.Context, driverID, nextCell string) error {
	driverCellKey := rediskeys.DriverH3CellKey(driverID)
	previousCell, err := s.rdb.Get(ctx, driverCellKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	if previousCell != "" && previousCell != nextCell {
		pipe.SRem(ctx, rediskeys.H3CellDriversKey(previousCell), driverID)
	}
	pipe.Set(ctx, driverCellKey, nextCell, s.geoTTL)
	pipe.SAdd(ctx, rediskeys.H3CellDriversKey(nextCell), driverID)
	_, err = pipe.Exec(ctx)
	return err
}
