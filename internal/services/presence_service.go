package services

import (
	"context"
	"strconv"
	"time"

	rediskeys "dispatch-socket-service/internal/redis"
	"dispatch-socket-service/internal/utils"

	"github.com/redis/go-redis/v9"
)

type PresenceService struct {
	rdb      redis.UniversalClient
	stateTTL time.Duration
}

func NewPresenceService(rdb redis.UniversalClient, stateTTL time.Duration) *PresenceService {
	return &PresenceService{rdb: rdb, stateTTL: stateTTL}
}

func (s *PresenceService) SetOnline(ctx context.Context, driverID string, isOnline bool) error {
	now := utils.NowUTC().Format(time.RFC3339)
	key := rediskeys.DriverStateKey(driverID)
	vals := map[string]interface{}{
		"driver_id":    driverID,
		"is_online":    strconv.FormatBool(isOnline),
		"last_seen_at": now,
	}
	if err := s.rdb.HSet(ctx, key, vals).Err(); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, key, s.stateTTL).Err()
}

func (s *PresenceService) SetAvailability(ctx context.Context, driverID string, isAvailable bool) error {
	key := rediskeys.DriverStateKey(driverID)
	vals := map[string]interface{}{
		"driver_id":    driverID,
		"is_available": strconv.FormatBool(isAvailable),
		"last_seen_at": utils.NowUTC().Format(time.RFC3339),
	}
	if err := s.rdb.HSet(ctx, key, vals).Err(); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, key, s.stateTTL).Err()
}

func (s *PresenceService) TouchLastSeen(ctx context.Context, driverID string) error {
	key := rediskeys.DriverStateKey(driverID)
	if err := s.rdb.HSet(ctx, key, "last_seen_at", utils.NowUTC().Format(time.RFC3339)).Err(); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, key, s.stateTTL).Err()
}
