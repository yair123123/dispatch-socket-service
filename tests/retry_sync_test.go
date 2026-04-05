package tests

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"dispatch-socket-service/internal/models"
	"dispatch-socket-service/internal/services"

	"github.com/stretchr/testify/require"
)

func TestCoreSyncRetryQueueBehavior(t *testing.T) {
	rdb, done := setupRedis(t)
	defer done()
	core := &fakeCoreClient{fail: true}
	svc := services.NewRetrySyncService(rdb, core, slog.Default(), 10*time.Millisecond, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.Start(ctx)

	err := svc.Enqueue(context.Background(), models.CoreAssignDriverRequest{RideID: "r1", DriverID: "d1", OfferID: "o1", RoundNumber: 1}, 0)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	require.GreaterOrEqual(t, core.calls, 2)
}
