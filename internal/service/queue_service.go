package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/port"
)

// QueueService handles queries against the maintenance queue.
//
// SAD Reference: Process Network 3 — "Consulta de Cola de Mantenimientos"
// Pattern: Service Layer (PoEAA)
type QueueService struct {
	repo   port.MaintenanceRepository
	logger *slog.Logger
}

// NewQueueService constructs a QueueService with its dependencies injected.
//
// Pattern: Dependency Injection (ADR-7)
func NewQueueService(repo port.MaintenanceRepository, logger *slog.Logger) *QueueService {
	return &QueueService{
		repo:   repo,
		logger: logger,
	}
}

// ListQueued retrieves all maintenance records currently in the queue.
//
// SAD Reference: Process Network 3 — "en cola"
func (s *QueueService) ListQueued(ctx context.Context) ([]*domain.Maintenance, error) {
	items, err := s.repo.ListByStatus(ctx, domain.MaintenanceStatusQueued)
	if err != nil {
		s.logger.ErrorContext(ctx, "maintenance_queue_query_failed",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("listing queued maintenances: %w", err)
	}
	return items, nil
}

// ListInProgress retrieves all maintenance records currently being processed.
//
// SAD Reference: Process Network 3 — "en mantenimiento"
func (s *QueueService) ListInProgress(ctx context.Context) ([]*domain.Maintenance, error) {
	items, err := s.repo.ListByStatus(ctx, domain.MaintenanceStatusInProgress)
	if err != nil {
		s.logger.ErrorContext(ctx, "maintenance_in_progress_query_failed",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("listing in-progress maintenances: %w", err)
	}
	return items, nil
}

// ListAll retrieves all maintenance records regardless of status.
func (s *QueueService) ListAll(ctx context.Context) ([]*domain.Maintenance, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "maintenance_list_query_failed",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("listing all maintenances: %w", err)
	}
	return items, nil
}

// GetByID retrieves a single maintenance record by ID.
func (s *QueueService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Maintenance, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.WarnContext(ctx, "maintenance_by_id_query_failed",
			slog.String("maintenance_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("getting maintenance %s: %w", id, err)
	}
	return m, nil
}
