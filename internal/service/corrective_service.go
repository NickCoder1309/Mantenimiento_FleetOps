package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/port"
)

// CorrectiveMaintenanceService handles the creation of corrective maintenance
// records triggered by the Incidents microservice.
//
// SAD Reference: Process Network 1 — "Registro de Mantenimiento Correctivo"
// Pattern: Service Layer (PoEAA), Dependency Injection
type CorrectiveMaintenanceService struct {
	repo   port.MaintenanceRepository
	logger *slog.Logger
}

// NewCorrectiveMaintenanceService constructs a CorrectiveMaintenanceService
// with its dependencies injected.
//
// Pattern: Dependency Injection (ADR-7)
func NewCorrectiveMaintenanceService(
	repo port.MaintenanceRepository,
	logger *slog.Logger,
) *CorrectiveMaintenanceService {
	return &CorrectiveMaintenanceService{
		repo:   repo,
		logger: logger,
	}
}

// CreateCorrective creates a new corrective maintenance record for a vehicle
// involved in an incident. The maintenance is immediately placed in the queue
// (status: queued) awaiting processing by the worker pool.
//
// SAD Reference: Process Network 1 — Steps 5-8
// Flow: Validate → Create domain entity → Persist via Repository → Return
func (s *CorrectiveMaintenanceService) CreateCorrective(
	ctx context.Context,
	vehicleID, incidentID uuid.UUID,
	severity uint8,
) (*domain.Maintenance, error) {
	// Step 5: Create the corrective maintenance domain entity
	maintenance, err := domain.NewCorrectiveMaintenance(vehicleID, incidentID, severity)
	if err != nil {
		s.logger.WarnContext(ctx, "maintenance_validation_failed",
			slog.String("vehicle_id", vehicleID.String()),
			slog.String("incident_id", incidentID.String()),
			slog.String("reason", err.Error()),
		)
		return nil, fmt.Errorf("creating corrective maintenance: %w", err)
	}

	// Steps 6-7: Persist via Repository → PostgreSQL
	if err := s.repo.Create(ctx, maintenance); err != nil {
		s.logger.ErrorContext(ctx, "maintenance_persistence_failed",
			slog.String("maintenance_id", maintenance.ID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("persisting corrective maintenance: %w", err)
	}

	s.logger.InfoContext(ctx, "maintenance_created",
		slog.String("maintenance_id", maintenance.ID.String()),
		slog.String("vehicle_id", vehicleID.String()),
		slog.String("incident_id", incidentID.String()),
		slog.String("user_id", ctx.),
		slog.Uint64("severity", uint64(severity)),
		slog.String("status", maintenance.Status.String()),
	)

	return maintenance, nil
}
