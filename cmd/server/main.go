// Package main is the application entry point and Composition Root.
// It wires all dependencies together following the Dependency Injection
// pattern and starts the HTTP server.
//
// Pattern: Composition Root
// SAD Reference: ADR-7 — "Dependency Injection: las dependencias son
// abstraídas mediante interfaces"
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fleetops/maintenance/internal/adapter/client"
	"github.com/fleetops/maintenance/internal/adapter/repository"
	"github.com/fleetops/maintenance/internal/handler"
	"github.com/fleetops/maintenance/internal/platform/config"
	"github.com/fleetops/maintenance/internal/platform/database"
	"github.com/fleetops/maintenance/internal/service"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize database connection pool
	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		os.Exit(1)
	}
	defer pool.Close()

	// =========================================================================
	// Dependency Injection — Composition Root
	// Wire: Adapters → Ports ← Services → Handlers
	// =========================================================================

	// Data Access Layer: Adapters (implement Port interfaces)
	maintenanceRepo := repository.NewPostgresMaintenanceRepository(pool)
	vehicleClient := client.NewHTTPVehicleClient(cfg.VehiclesServiceURL, cfg.HTTPClientTimeoutSecs)

	// Business Logic Layer: Services (depend on Port interfaces)
	correctiveSvc := service.NewCorrectiveMaintenanceService(maintenanceRepo)
	preventiveSvc := service.NewPreventiveMaintenanceService(
		maintenanceRepo, vehicleClient,
		cfg.PreventiveKmThreshold, cfg.PreventiveDaysThreshold,
		cfg.CronIntervalDays,
	)
	queueSvc := service.NewQueueService(maintenanceRepo)
	workerPool := service.NewWorkerPool(
		maintenanceRepo, vehicleClient,
		cfg.MaxWorkers, cfg.WorkerPollIntervalSecs,
	)

	// Presentation Layer: Handlers (depend on Services)
	maintenanceHandler := handler.NewMaintenanceHandler(correctiveSvc, queueSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// Router
	router := handler.NewRouter(maintenanceHandler, healthHandler, cfg.MetricsEnabled)

	// =========================================================================
	// Start background services
	// =========================================================================

	// Start preventive maintenance scheduler (Cron Handler)
	preventiveSvc.Start(ctx)
	defer preventiveSvc.Stop()

	// Start worker pool (Bulkhead pattern)
	workerPool.Start(ctx)
	defer workerPool.Stop()

	// =========================================================================
	// HTTP Server with graceful shutdown
	// =========================================================================

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			os.Exit(1)
		}
	}()

	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		os.Exit(1)
	}
}
