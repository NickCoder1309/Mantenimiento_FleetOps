package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/handler"
	"github.com/fleetops/maintenance/internal/handler/dto"
	"github.com/fleetops/maintenance/internal/mocks"
	"github.com/fleetops/maintenance/internal/service"
)

func setupHandler() (*handler.MaintenanceHandler, *mocks.MockMaintenanceRepository) {
	repo := new(mocks.MockMaintenanceRepository)
	correctiveSvc := service.NewCorrectiveMaintenanceService(repo)
	queueSvc := service.NewQueueService(repo)
	h := handler.NewMaintenanceHandler(correctiveSvc, queueSvc)
	return h, repo
}

// =============================================================================
// CreateCorrective handler tests
// =============================================================================

func TestCreateCorrective_Handler_Success(t *testing.T) {
	// Arrange
	h, repo := setupHandler()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Maintenance")).Return(nil)

	body := dto.CreateMaintenanceRequest{
		VehicleID:  uuid.New(),
		IncidentID: uuid.New(),
		Severity:   5,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mantenimientos", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Act
	h.CreateCorrective(rr, req)

	// Assert
	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp dto.MaintenanceResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, body.VehicleID, resp.VehicleID)
	assert.Equal(t, "corrective", resp.Type)
	assert.Equal(t, "queued", resp.Status)
	repo.AssertExpectations(t)
}

func TestCreateCorrective_Handler_InvalidJSON(t *testing.T) {
	// Arrange
	h, _ := setupHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mantenimientos",
		bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()

	// Act
	h.CreateCorrective(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateCorrective_Handler_ValidationError(t *testing.T) {
	// Arrange
	h, _ := setupHandler()

	body := dto.CreateMaintenanceRequest{
		VehicleID:  uuid.Nil, // invalid
		IncidentID: uuid.New(),
		Severity:   5,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mantenimientos",
		bytes.NewReader(jsonBody))
	rr := httptest.NewRecorder()

	// Act
	h.CreateCorrective(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateCorrective_Handler_SeverityValidation(t *testing.T) {
	// Arrange
	h, _ := setupHandler()

	body := dto.CreateMaintenanceRequest{
		VehicleID:  uuid.New(),
		IncidentID: uuid.New(),
		Severity:   0, // invalid
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mantenimientos",
		bytes.NewReader(jsonBody))
	rr := httptest.NewRecorder()

	// Act
	h.CreateCorrective(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// =============================================================================
// ListAll handler tests
// =============================================================================

func TestListAll_Handler_Success(t *testing.T) {
	// Arrange
	h, repo := setupHandler()

	items := []*domain.Maintenance{
		{ID: uuid.New(), VehicleID: uuid.New(), Type: domain.MaintenanceTypeCorrective, Status: domain.MaintenanceStatusQueued},
	}
	repo.On("List", mock.Anything).Return(items, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mantenimientos", nil)
	rr := httptest.NewRecorder()

	// Act
	h.ListAll(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp []*dto.MaintenanceResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp, 1)
}

func TestListAll_Handler_RepositoryError(t *testing.T) {
	// Arrange
	h, repo := setupHandler()

	repo.On("List", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mantenimientos", nil)
	rr := httptest.NewRecorder()

	// Act
	h.ListAll(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// =============================================================================
// GetByID handler tests
// =============================================================================

func TestGetByID_Handler_Success(t *testing.T) {
	// Arrange
	h, repo := setupHandler()

	id := uuid.New()
	expected := &domain.Maintenance{ID: id, VehicleID: uuid.New(), Status: domain.MaintenanceStatusQueued}
	repo.On("GetByID", mock.Anything, id).Return(expected, nil)

	// Use chi context to inject URL param
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mantenimientos/"+id.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	// Simpler: set chi URL param directly
	rr := httptest.NewRecorder()

	// Act
	h.GetByID(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGetByID_Handler_InvalidUUID(t *testing.T) {
	// Arrange
	h, _ := setupHandler()

	req := setChiURLParam(
		httptest.NewRequest(http.MethodGet, "/api/v1/mantenimientos/not-a-uuid", nil),
		"id", "not-a-uuid",
	)
	rr := httptest.NewRecorder()

	// Act
	h.GetByID(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetByID_Handler_NotFound(t *testing.T) {
	// Arrange
	h, repo := setupHandler()

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrMaintenanceNotFound)

	req := setChiURLParam(
		httptest.NewRequest(http.MethodGet, "/api/v1/mantenimientos/"+id.String(), nil),
		"id", id.String(),
	)
	rr := httptest.NewRecorder()

	// Act
	h.GetByID(rr, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// setChiURLParam is a test helper to inject chi URL parameters.
func setChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
