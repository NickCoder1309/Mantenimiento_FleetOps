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
// CreateCorrective tests
// =============================================================================

func TestCreateCorrective_Success(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewCorrectiveMaintenanceService(repo)

	vehicleID := uuid.New()
	incidentID := uuid.New()
	severity := uint8(5)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).Return(nil)

	// Act
	m, err := svc.CreateCorrective(context.Background(), vehicleID, incidentID, severity)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, vehicleID, m.VehicleID)
	assert.Equal(t, &incidentID, m.IncidentID)
	assert.Equal(t, domain.MaintenanceTypeCorrective, m.Type)
	assert.Equal(t, severity, m.Severity)
	assert.Equal(t, domain.MaintenanceStatusQueued, m.Status)
	repo.AssertExpectations(t)
}

func TestCreateCorrective_InvalidVehicleID(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewCorrectiveMaintenanceService(repo)

	// Act
	m, err := svc.CreateCorrective(context.Background(), uuid.Nil, uuid.New(), 5)

	// Assert
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidVehicleID)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateCorrective_InvalidIncidentID(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewCorrectiveMaintenanceService(repo)

	// Act
	m, err := svc.CreateCorrective(context.Background(), uuid.New(), uuid.Nil, 5)

	// Assert
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidIncidentID)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateCorrective_InvalidSeverity(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewCorrectiveMaintenanceService(repo)

	// Act
	m, err := svc.CreateCorrective(context.Background(), uuid.New(), uuid.New(), 0)

	// Assert
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidSeverity)
	repo.AssertNotCalled(t, "Create")
}

func TestCreateCorrective_RepositoryError(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewCorrectiveMaintenanceService(repo)

	dbErr := errors.New("database connection lost")
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).Return(dbErr)

	// Act
	m, err := svc.CreateCorrective(context.Background(), uuid.New(), uuid.New(), 5)

	// Assert
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	repo.AssertExpectations(t)
}
