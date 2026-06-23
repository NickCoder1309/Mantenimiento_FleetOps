package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fleetops/maintenance/internal/domain"
	"github.com/fleetops/maintenance/internal/port"
)

// PreventiveMaintenanceService handles the automatic scheduling of preventive
// maintenance records based on vehicle usage thresholds.
//
// SAD Reference: Process Network 2 — "Programación Automática de Mantenimientos
// Preventivos"
// Pattern: Service Layer (PoEAA), Scheduled Task (Cron Handler)
type PreventiveMaintenanceService struct {
	repo            port.MaintenanceRepository
	vehicleClient   port.VehicleClient
	kmThreshold     float64
	daysThreshold   int
	intervalDays    int
	stopCh          chan struct{}
	stopped         sync.Once
}

// NewPreventiveMaintenanceService constructs a PreventiveMaintenanceService
// with all dependencies injected.
//
// Pattern: Dependency Injection (ADR-7)
func NewPreventiveMaintenanceService(
	repo port.MaintenanceRepository,
	vehicleClient port.VehicleClient,
	kmThreshold float64,
	daysThreshold int,
	intervalDays int,
) *PreventiveMaintenanceService {
	return &PreventiveMaintenanceService{
		repo:          repo,
		vehicleClient: vehicleClient,
		kmThreshold:   kmThreshold,
		daysThreshold: daysThreshold,
		intervalDays:  intervalDays,
		stopCh:        make(chan struct{}),
	}
}

// SchedulePreventive executes a single preventive maintenance scheduling cycle.
// It fetches all vehicles, filters those needing maintenance, and creates
// preventive maintenance records for each qualifying vehicle.
//
// SAD Reference: Process Network 2 — Steps 1-7
// Flow: GET /vehiculos → Filter → Generate maintenances → Persist
func (s *PreventiveMaintenanceService) SchedulePreventive(ctx context.Context) ([]*domain.Maintenance, error) {
	// Steps 2-3: Fetch vehicles from external service via ACL
	vehicles, err := s.vehicleClient.GetAllVehicles(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching vehicles: %w", err)
	}

	// Step 4: Filter vehicles based on thresholds
	var created []*domain.Maintenance
	for _, v := range vehicles {
		if !v.NeedsPreventiveMaintenance(s.kmThreshold, s.daysThreshold) {
			continue
		}

		// Step 5: Generate preventive maintenance record
		m, err := domain.NewPreventiveMaintenance(v.ID)
		if err != nil {
			continue
		}

		// Steps 6-7: Persist via Repository
		if err := s.repo.Create(ctx, m); err != nil {
			continue
		}

		created = append(created, m)
	}

	return created, nil
}

// Start begins the periodic preventive maintenance scheduling loop.
// It runs in a separate goroutine and executes SchedulePreventive every
// intervalDays days.
//
// SAD Reference: "Cron Handler se ejecuta cada X días"
func (s *PreventiveMaintenanceService) Start(ctx context.Context) {
	interval := time.Duration(s.intervalDays) * 24 * time.Hour
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.SchedulePreventive(ctx); err != nil {
				}
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop signals the scheduler to stop its periodic execution.
func (s *PreventiveMaintenanceService) Stop() {
	s.stopped.Do(func() {
		close(s.stopCh)
	})
}
