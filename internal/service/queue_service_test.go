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
// ListQueued tests
// =============================================================================

func TestListQueued_Success(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	expected := []*domain.Maintenance{
		{ID: uuid.New(), Status: domain.MaintenanceStatusQueued},
		{ID: uuid.New(), Status: domain.MaintenanceStatusQueued},
	}
	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatusQueued).Return(expected, nil)

	// Act
	result, err := svc.ListQueued(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestListQueued_RepositoryError(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatusQueued).
		Return(nil, errors.New("db error"))

	// Act
	result, err := svc.ListQueued(context.Background())

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
}

// =============================================================================
// ListInProgress tests
// =============================================================================

func TestListInProgress_Success(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	expected := []*domain.Maintenance{
		{ID: uuid.New(), Status: domain.MaintenanceStatusInProgress},
	}
	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatusInProgress).Return(expected, nil)

	// Act
	result, err := svc.ListInProgress(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestListInProgress_RepositoryError(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	repo.On("ListByStatus", mock.Anything, domain.MaintenanceStatusInProgress).
		Return(nil, errors.New("db error"))

	// Act
	result, err := svc.ListInProgress(context.Background())

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
}

// =============================================================================
// ListAll tests
// =============================================================================

func TestListAll_Success(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	expected := []*domain.Maintenance{
		{ID: uuid.New(), Status: domain.MaintenanceStatusQueued},
		{ID: uuid.New(), Status: domain.MaintenanceStatusCompleted},
	}
	repo.On("List", mock.Anything).Return(expected, nil)

	// Act
	result, err := svc.ListAll(context.Background())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestListAll_RepositoryError(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	repo.On("List", mock.Anything).Return(nil, errors.New("db error"))

	// Act
	result, err := svc.ListAll(context.Background())

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
}

// =============================================================================
// GetByID tests
// =============================================================================

func TestGetByID_Success(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	id := uuid.New()
	expected := &domain.Maintenance{ID: id, Status: domain.MaintenanceStatusQueued}
	repo.On("GetByID", mock.Anything, id).Return(expected, nil)

	// Act
	result, err := svc.GetByID(context.Background(), id)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetByID_NotFound(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrMaintenanceNotFound)

	// Act
	result, err := svc.GetByID(context.Background(), id)

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrMaintenanceNotFound)
}

func TestGetByID_RepositoryError(t *testing.T) {
	// Arrange
	repo := new(mocks.MockMaintenanceRepository)
	svc := service.NewQueueService(repo)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("db error"))

	// Act
	result, err := svc.GetByID(context.Background(), id)

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
}
