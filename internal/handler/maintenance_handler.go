package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/handler/dto"
	"github.com/fleetops/maintenance/internal/service"
)

// MaintenanceHandler handles HTTP requests for maintenance operations.
//
// SAD Reference: "HTTP Handler valida los datos recibidos"
// Pattern: Handler (Presentation Layer)
type MaintenanceHandler struct {
	correctiveSvc *service.CorrectiveMaintenanceService
	queueSvc      *service.QueueService
}

// NewMaintenanceHandler constructs a MaintenanceHandler with injected services.
func NewMaintenanceHandler(
	correctiveSvc *service.CorrectiveMaintenanceService,
	queueSvc *service.QueueService,
) *MaintenanceHandler {
	return &MaintenanceHandler{
		correctiveSvc: correctiveSvc,
		queueSvc:      queueSvc,
	}
}

// CreateCorrective handles POST /api/v1/mantenimientos
// This is the primary transactional flow entry point (Rule R2).
//
// SAD Reference: Process Network 1 — Steps 3-9
// Flow: Receive HTTP → Validate DTO → Delegate to Service → Return Response
func (h *MaintenanceHandler) CreateCorrective(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMaintenanceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	// Step 4: HTTP Handler validates the received data
	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Step 5-8: Delegate to service layer
	maintenance, err := h.correctiveSvc.CreateCorrective(
		r.Context(),
		req.VehicleID,
		req.IncidentID,
		req.Severity,
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidVehicleID) ||
			errors.Is(err, domain.ErrInvalidIncidentID) ||
			errors.Is(err, domain.ErrInvalidSeverity) {
			h.respondError(w, http.StatusBadRequest, "domain_validation_error", err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, "creation_failed", "Failed to create maintenance")
		return
	}

	// Step 9: Return confirmation
	h.respondJSON(w, http.StatusCreated, dto.FromDomain(maintenance))
}

// ListAll handles GET /api/v1/mantenimientos
//
// SAD Reference: Process Network 3 — "Consulta de Cola de Mantenimientos"
func (h *MaintenanceHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.queueSvc.ListAll(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "list_failed", "Failed to retrieve maintenances")
		return
	}

	h.respondJSON(w, http.StatusOK, dto.FromDomainList(items))
}

// GetByID handles GET /api/v1/mantenimientos/{id}
func (h *MaintenanceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid UUID")
		return
	}

	m, err := h.queueSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrMaintenanceNotFound) {
			h.respondError(w, http.StatusNotFound, "not_found", "Maintenance not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "get_failed", "Failed to retrieve maintenance")
		return
	}

	h.respondJSON(w, http.StatusOK, dto.FromDomain(m))
}

// GetQueueSummary handles GET /api/v1/mantenimientos/cola
// Returns the maintenance queue summary: queued + in-progress items.
//
// SAD Reference: Process Network 3 — "listado de vehículos: en cola, en mantenimiento"
func (h *MaintenanceHandler) GetQueueSummary(w http.ResponseWriter, r *http.Request) {
	queued, err := h.queueSvc.ListQueued(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "queue_failed", "Failed to retrieve queue")
		return
	}

	inProgress, err := h.queueSvc.ListInProgress(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "queue_failed", "Failed to retrieve in-progress")
		return
	}

	summary := dto.QueueSummaryResponse{
		Queued:      dto.FromDomainList(queued),
		InProgress:  dto.FromDomainList(inProgress),
		TotalQueued: len(queued),
		TotalActive: len(inProgress),
	}

	h.respondJSON(w, http.StatusOK, summary)
}

// respondJSON writes a JSON response with the given status code.
func (h *MaintenanceHandler) respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// respondError writes a structured error response.
func (h *MaintenanceHandler) respondError(w http.ResponseWriter, code int, errType, message string) {
	h.respondJSON(w, code, dto.ErrorResponse{
		Error:   errType,
		Message: message,
		Code:    code,
	})
}
