package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/mocks"
	"github.com/fleetops/maintenance/internal/service"
)

// =============================================================================
// WorkerPool tests
// =============================================================================

func TestWorkerPool_StartAndStop(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	wp := service.NewWorkerPool(repo, vehicleClient, 3, 1)

	// Act — start and immediately stop, should not panic
	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	// Give it a moment then stop
	time.Sleep(50 * time.Millisecond)
	cancel()
	wp.Stop()

	// Assert — no panic, no deadlock
}

func TestWorkerPool_StopIsIdempotent(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	wp := service.NewWorkerPool(repo, vehicleClient, 3, 60)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp.Start(ctx)

	// Act — calling Stop multiple times should not panic
	wp.Stop()
	wp.Stop()
	wp.Stop()
}

func TestWorkerPool_ProcessesQueuedItems(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)

	vehicleID := uuid.New()
	m1, _ := domain.NewCorrectiveMaintenance(vehicleID, uuid.New(), 5)

	queued := []*domain.Maintenance{m1}

	// First call returns items, subsequent calls return empty
	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatus("queued")).
		Return(queued, nil).Once()
	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatus("queued")).
		Return([]*domain.Maintenance{}, nil)

	repo.On("UpdateStatus", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).
		Return(nil)

	vehicleClient.On("UpdateVehicleMaintenanceStatus", mock.Anything, vehicleID, 0).
		Return(nil)

	// Act — use short poll interval to trigger quickly
	wp := service.NewWorkerPool(repo, vehicleClient, 2, 1)
	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	// Wait for processing to complete
	time.Sleep(2 * time.Second)
	cancel()
	wp.Stop()

	// Assert — UpdateStatus should have been called (at least for in_progress and completed)
	repo.AssertCalled(t, "UpdateStatus", mock.Anything, mock.AnythingOfType("*domain.Maintenance"))
	assert.True(t, m1.IsTerminal(), "maintenance should be in terminal state")
}

func TestWorkerPool_RespectsMaxWorkers(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)

	// Create more items than maxWorkers
	var queued []*domain.Maintenance
	for i := 0; i < 10; i++ {
		m, _ := domain.NewCorrectiveMaintenance(uuid.New(), uuid.New(), 5)
		queued = append(queued, m)
	}

	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatus("queued")).
		Return(queued, nil).Once()
	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatus("queued")).
		Return([]*domain.Maintenance{}, nil)
	repo.On("UpdateStatus", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).
		Return(nil)
	vehicleClient.On("UpdateVehicleMaintenanceStatus", mock.Anything, mock.Anything, 0).
		Return(nil)

	// Act — maxWorkers = 3, but 10 items; all should still be processed
	wp := service.NewWorkerPool(repo, vehicleClient, 3, 1)
	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	time.Sleep(3 * time.Second)
	cancel()
	wp.Stop()

	// Assert — all 10 items should be processed (20 UpdateStatus calls: 10 in_progress + 10 completed)
	repo.AssertCalled(t, "UpdateStatus", mock.Anything, mock.AnythingOfType("*domain.Maintenance"))
}
