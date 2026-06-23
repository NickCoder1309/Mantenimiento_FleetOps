package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/mocks"
	"github.com/fleetops/maintenance/internal/service"
)

// =============================================================================
// SchedulePreventive tests
// =============================================================================

func TestSchedulePreventive_Success_FiltersAndCreates(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	svc := service.NewPreventiveMaintenanceService(
		repo, vehicleClient, 10000, 90, 7,
	)

	vehicles := []*domain.Vehicle{
		{ID: uuid.New(), KilometersRecorded: 15000, DaysSinceLastMaintenance: 30, Available: true},  // qualifies (km)
		{ID: uuid.New(), KilometersRecorded: 5000, DaysSinceLastMaintenance: 100, Available: true},  // qualifies (days)
		{ID: uuid.New(), KilometersRecorded: 3000, DaysSinceLastMaintenance: 20, Available: true},   // does NOT qualify
		{ID: uuid.New(), KilometersRecorded: 20000, DaysSinceLastMaintenance: 120, Available: false}, // NOT available
	}

	vehicleClient.On("GetAllVehicles", mock.Anything).Return(vehicles, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).Return(nil)

	// Act
	created, err := svc.SchedulePreventive(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Len(t, created, 2) // only 2 vehicles qualify
	for _, m := range created {
		assert.Equal(t, domain.MaintenanceTypePreventive, m.Type)
		assert.Equal(t, domain.MaintenanceStatusQueued, m.Status)
	}
	vehicleClient.AssertExpectations(t)
	repo.AssertNumberOfCalls(t, "Create", 2)
}

func TestSchedulePreventive_NoVehiclesQualify(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	svc := service.NewPreventiveMaintenanceService(
		repo, vehicleClient, 10000, 90, 7,
	)

	vehicles := []*domain.Vehicle{
		{ID: uuid.New(), KilometersRecorded: 5000, DaysSinceLastMaintenance: 30, Available: true},
	}

	vehicleClient.On("GetAllVehicles", mock.Anything).Return(vehicles, nil)

	// Act
	created, err := svc.SchedulePreventive(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Empty(t, created)
	repo.AssertNotCalled(t, "Create")
}

func TestSchedulePreventive_VehicleClientError(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	svc := service.NewPreventiveMaintenanceService(
		repo, vehicleClient, 10000, 90, 7,
	)

	vehicleClient.On("GetAllVehicles", mock.Anything).Return(nil, errors.New("connection refused"))

	// Act
	created, err := svc.SchedulePreventive(context.Background())

	// Assert
	assert.Nil(t, created)
	assert.Error(t, err)
	repo.AssertNotCalled(t, "Create")
}

func TestSchedulePreventive_RepositoryError_ContinuesProcessing(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	svc := service.NewPreventiveMaintenanceService(
		repo, vehicleClient, 10000, 90, 7,
	)

	vehicles := []*domain.Vehicle{
		{ID: uuid.New(), KilometersRecorded: 15000, DaysSinceLastMaintenance: 30, Available: true},
		{ID: uuid.New(), KilometersRecorded: 20000, DaysSinceLastMaintenance: 30, Available: true},
	}

	vehicleClient.On("GetAllVehicles", mock.Anything).Return(vehicles, nil)
	// First call fails, second succeeds
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).
		Return(errors.New("db error")).Once()
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).
		Return(nil).Once()

	// Act
	created, err := svc.SchedulePreventive(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Len(t, created, 1) // only the second one succeeded
	repo.AssertNumberOfCalls(t, "Create", 2)
}

func TestSchedulePreventive_EmptyVehicleList(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	vehicleClient := new(mocks.MockVehicleClient)
	svc := service.NewPreventiveMaintenanceService(
		repo, vehicleClient, 10000, 90, 7,
	)

	vehicleClient.On("GetAllVehicles", mock.Anything).Return([]*domain.Vehicle{}, nil)

	// Act
	created, err := svc.SchedulePreventive(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Empty(t, created)
}
