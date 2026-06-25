# FleetOps Maintenance Microservice

Microservicio de mantenimiento de la plataforma **FleetOps**, responsable de gestionar mantenimientos preventivos y correctivos de vehículos, administrar la cola de mantenimientos y coordinar el procesamiento concurrente de tareas.

**Arquitectura**: Microservicios con separación por capas (Strict Layering), patrón Repository, Bulkhead para concurrencia limitada, y Dependency Injection para testeabilidad.

---

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| **Go** | >= 1.22.0 | Language runtime |
| **Docker** | >= 27.0 | Containerization |
| **Docker Compose** | >= 2.20 (V2) | Local orchestration |
| **golang-migrate** | >= 4.17 | Database migrations (optional, local) |
| **golangci-lint** | >= 1.59 | Linting (optional, local) |
| **gofumpt** | latest | Code formatting (optional, local) |

---

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/fleetops/maintenance.git
cd maintenance

# 2. Copy environment template
cp .env.example .env

# 3. Edit .env with your configuration (default values work for local Docker)

# 4. Start all services (PostgreSQL, Migrations, Maintenance, Prometheus, Grafana)
docker compose up --build

# 5. Verify the service is running (migrations apply automatically on startup)
curl http://localhost:8080/health
```

The service will be available at:
- **API**: `http://localhost:8080/api/v1/mantenimientos`
- **Health**: `http://localhost:8080/health`
- **Metrics**: `http://localhost:8080/metrics`
---

## API Endpoints

| Method | Path | Description | SAD Reference |
|---|---|---|---|
| `POST` | `/api/v1/mantenimientos` | Create corrective maintenance | Process Network 1 |
| `GET` | `/api/v1/mantenimientos` | List all maintenances | Process Network 3 |
| `GET` | `/api/v1/mantenimientos/{id}` | Get maintenance by ID | — |
| `GET` | `/api/v1/mantenimientos/cola` | Queue summary (queued + in-progress) | Process Network 3 |
| `GET` | `/health` | Health check | Convention |

### Example: Create Corrective Maintenance

```bash
curl -X POST http://localhost:8080/api/v1/mantenimientos \
  -H "Content-Type: application/json" \
  -d '{
    "id_incidente": "550e8400-e29b-41d4-a716-446655440001",
    "id_vehiculo": "550e8400-e29b-41d4-a716-446655440002",
    "gravedad": 7
  }'
```

---

## Running Tests Locally (without Docker)

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v -race ./internal/...
```

---

## Coverage Report

```bash
# Generate coverage report (domain + service + handler layers)
make test-coverage

# The HTML report will be at: coverage/coverage.html
# The console will display function-level coverage percentages
```

The coverage tool is configured to enforce 100% coverage on the `domain` and `service` layers (application logic). Infrastructure adapters requiring real database connections are excluded from unit test coverage (covered by integration tests).

---

## Project Structure

```
fleetops-maintenance/
├── cmd/server/main.go ················ Entry Point (Composition Root)
├── internal/
│   ├── domain/ ······················· Business Logic Layer — Domain Models
│   │   ├── maintenance.go ··········· Maintenance entity (state machine)
│   │   ├── vehicle.go ··············· Vehicle ACL value object
│   │   └── errors.go ················ Domain error definitions
│   ├── port/ ························· Business Logic Layer — Ports (interfaces)
│   │   ├── repository.go ············ MaintenanceRepository interface
│   │   └── vehicle_client.go ········ VehicleClient ACL interface
│   ├── service/ ····················· Business Logic Layer — Application Services
│   │   ├── corrective_service.go ···· Corrective maintenance (Process Network 1)
│   │   ├── preventive_service.go ···· Preventive scheduling (Process Network 2)
│   │   ├── queue_service.go ········· Queue queries (Process Network 3)
│   │   └── worker_pool.go ··········· Concurrent processing (Bulkhead pattern)
│   ├── adapter/ ····················· Data Access Layer — Adapters
│   │   ├── repository/ ·············· PostgreSQL repository (pgx v5)
│   │   └── client/ ················· HTTP vehicle client (ACL)
│   ├── handler/ ····················· Presentation Layer — HTTP Handlers
│   │   ├── router.go ················ Chi router setup
│   │   ├── maintenance_handler.go ··· REST endpoint handlers
│   │   ├── health_handler.go ········ Health check handler
│   │   ├── dto/ ····················· Request/Response DTOs
│   │   └── middleware/ ·············· Recovery middleware
│   ├── platform/ ···················· Cross-Cutting Concerns
│   │   ├── config/ ·················· Environment configuration
│   │   ├── database/ ················ PostgreSQL connection pool
│   └── mocks/ ······················· Test mocks (testify/mock)
├── migrations/ ······················ Database schema migrations
├── .github/workflows/ci.yml ·········· CI pipeline (lint, test, build)
├── Dockerfile ························ Multi-stage build
├── docker-compose.yml ················ Local dev orchestration
├── prometheus.yml ···················· Prometheus scrape config
├── grafana/ ·························· Grafana provisioning
├── .env.example ······················ Environment template
├── .golangci.yml ····················· Linter configuration
├── Makefile ·························· Build automation
└── README.md ························· This file
```

---

## Key Architectural Decisions

| ID | Decision | Pattern | SAD Reference |
|---|---|---|---|
| ADR-1 | Go (Golang) as primary language | — | Concurrency via goroutines |
| ADR-2 | PostgreSQL via Supabase (pgx v5) | Database Per Service | Data isolation |
| ADR-3 | Three-layer strict architecture | Layered Architecture | Presentation → Logic → Data |
| ADR-4 | Repository for persistence abstraction | Repository (PoEAA) | Decoupling |
| ADR-5 | Bulkhead for concurrent workers | Bulkhead (Resilience) | Fault isolation |
| ADR-7 | Dependency Injection via interfaces | DI (SOLID) | Testability |
| ADR-10 | Prometheus + Grafana | Centralized Monitoring | Observability |
| ADR-11 | API Gateway (external) | API Gateway | Routing & auth |
| Conv. | golang-migrate | Migration Management | Schema evolution |
| Conv. | Anti-Corruption Layer | ACL (DDD) | External service isolation |

---

## Known Limitations and Recommended Next Steps

### Limitations
1. **No authentication/authorization** — Handled externally by the API Gateway (ADR-11). The microservice trusts all requests that reach it.
2. **No message broker** — All communication is synchronous REST. For high-throughput scenarios, consider adding an event bus (RabbitMQ, Kafka).
3. **No circuit breaker** — External service calls (Vehicles) lack circuit breaker protection. Consider adding `sony/gobreaker` or similar.
4. **Worker pool processing is simulated** — The actual maintenance processing logic (what happens during a maintenance) is a placeholder. Real business logic should be implemented.
5. **Supabase-specific features unused** — Since we use raw pgx, Supabase-specific features (real-time subscriptions, auth, storage) are not leveraged.

### Recommended Next Steps
1. **Integration tests** — Add `testcontainers-go` for PostgreSQL integration tests of the repository layer.
2. **Circuit Breaker** — Implement `CircuitBreaker` pattern for external HTTP calls.
3. **Rate limiting** — Add rate limiting middleware for public endpoints.
4. **API documentation** — Generate OpenAPI/Swagger spec from handler annotations.
5. **Database connection retry** — Add exponential backoff for database connection at startup.
6. **Distributed tracing** — Integrate OpenTelemetry for distributed tracing across microservices.
7. **AWS ECS task definition** — Create ECS task definition and service configuration for production deployment (FC-11).
