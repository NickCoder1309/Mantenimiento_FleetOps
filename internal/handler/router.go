package handler

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fleetops/maintenance/internal/handler/middleware"
)

// NewRouter creates and configures the Chi router with all routes and middleware.
//
// SAD Reference: "Router HTTP" — receives and dispatches HTTP requests
// Pattern: Router (Integration Pattern)
// Tech: Chi + net/http (stdlib) — confirmed decision FC-1
//
// Routes:
//
//	POST   /api/v1/mantenimientos        → CreateCorrective (Process Network 1)
//	GET    /api/v1/mantenimientos        → ListAll          (Process Network 3)
//	GET    /api/v1/mantenimientos/{id}   → GetByID
//	GET    /api/v1/mantenimientos/cola   → GetQueueSummary  (Process Network 3)
//	GET    /health                       → Health Check
//	GET    /metrics                      → Prometheus Metrics (ADR-10)
func NewRouter(
	maintenanceHandler *MaintenanceHandler,
	healthHandler *HealthHandler,
	metricsEnabled bool,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Recovery())

	// Health check — outside versioned API
	r.Get("/health", healthHandler.Check)

	// Prometheus metrics endpoint
	// SAD Reference: ADR-10 — Prometheus for metric collection
	if metricsEnabled {
		r.Handle("/metrics", promhttp.Handler())
	}

	// Versioned API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/mantenimientos", func(r chi.Router) {
			// Process Network 1: Create corrective maintenance
			r.Post("/", maintenanceHandler.CreateCorrective)

			// Process Network 3: Query maintenance queue
			r.Get("/", maintenanceHandler.ListAll)
			r.Get("/cola", maintenanceHandler.GetQueueSummary)
			r.Get("/{id}", maintenanceHandler.GetByID)
		})
	})

	return r
}
