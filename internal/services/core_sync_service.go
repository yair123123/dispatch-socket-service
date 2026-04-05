package services

import (
	"context"
	"log/slog"

	"dispatch-socket-service/internal/clients"
	"dispatch-socket-service/internal/models"
)

type CoreSyncService struct {
	coreClient clients.CoreClient
	retrySync  *RetrySyncService
	logger     *slog.Logger
}

func NewCoreSyncService(coreClient clients.CoreClient, retrySync *RetrySyncService, logger *slog.Logger) *CoreSyncService {
	return &CoreSyncService{coreClient: coreClient, retrySync: retrySync, logger: logger}
}

func (s *CoreSyncService) SyncAssignment(ctx context.Context, req models.CoreAssignDriverRequest) {
	if err := s.coreClient.AssignDriver(ctx, req); err != nil {
		s.logger.Error("core sync failed, enqueue retry", "ride_id", req.RideID, "driver_id", req.DriverID, "error", err)
		if enqueueErr := s.retrySync.Enqueue(ctx, req, 0); enqueueErr != nil {
			s.logger.Error("enqueue retry failed", "ride_id", req.RideID, "driver_id", req.DriverID, "error", enqueueErr)
		}
		return
	}
	s.logger.Info("core sync success", "ride_id", req.RideID, "driver_id", req.DriverID)
}
