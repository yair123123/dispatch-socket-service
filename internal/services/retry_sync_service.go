package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"dispatch-socket-service/internal/clients"
	"dispatch-socket-service/internal/models"
	rediskeys "dispatch-socket-service/internal/redis"

	"github.com/redis/go-redis/v9"
)

type RetrySyncService struct {
	rdb        redis.UniversalClient
	coreClient clients.CoreClient
	logger     *slog.Logger
	interval   time.Duration
	maxRetries int
	stopCh     chan struct{}
}

type retryJob struct {
	Request models.CoreAssignDriverRequest `json:"request"`
	Retries int                            `json:"retries"`
}

func NewRetrySyncService(rdb redis.UniversalClient, coreClient clients.CoreClient, logger *slog.Logger, interval time.Duration, maxRetries int) *RetrySyncService {
	return &RetrySyncService{rdb: rdb, coreClient: coreClient, logger: logger, interval: interval, maxRetries: maxRetries, stopCh: make(chan struct{})}
}

func (s *RetrySyncService) Enqueue(ctx context.Context, req models.CoreAssignDriverRequest, retries int) error {
	payload, err := json.Marshal(retryJob{Request: req, Retries: retries})
	if err != nil {
		return err
	}
	return s.rdb.RPush(ctx, rediskeys.CoreSyncRetryQueueKey, payload).Err()
}

func (s *RetrySyncService) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processOne(ctx)
		}
	}
}

func (s *RetrySyncService) Stop() { close(s.stopCh) }

func (s *RetrySyncService) processOne(ctx context.Context) {
	payload, err := s.rdb.LPop(ctx, rediskeys.CoreSyncRetryQueueKey).Result()
	if err == redis.Nil {
		return
	}
	if err != nil {
		s.logger.Error("retry lpop failed", "error", err)
		return
	}
	var job retryJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		s.logger.Error("retry unmarshal failed", "error", err)
		return
	}
	if err := s.coreClient.AssignDriver(ctx, job.Request); err != nil {
		job.Retries++
		if job.Retries <= s.maxRetries {
			s.logger.Warn("core sync retry failed, requeue", "ride_id", job.Request.RideID, "driver_id", job.Request.DriverID, "retries", job.Retries, "error", err)
			_ = s.Enqueue(ctx, job.Request, job.Retries)
		} else {
			s.logger.Error("core sync dropped after max retries", "ride_id", job.Request.RideID, "driver_id", job.Request.DriverID, "error", err)
		}
		return
	}
	s.logger.Info("core sync retry success", "ride_id", job.Request.RideID, "driver_id", job.Request.DriverID)
}
