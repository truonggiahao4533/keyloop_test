# Clean Architecture Implementation Skill

## When to Use This Skill

Apply this skill whenever creating, modifying, or extending features in this repository. This document defines the **canonical project structure**, **layer responsibilities**, **dependency flow**, and **file naming conventions** that MUST be followed.

---

## Architecture Overview

This project follows **Clean Architecture** (Ports & Adapters / Hexagonal Architecture), ensuring:
- **Domain** is at the center with zero external dependencies.
- **Use cases** orchestrate business logic, depending only on domain and repository/provider interfaces.
- **Infrastructure** implements the interfaces defined by inner layers.
- **Dependency flow is inward** — outer layers depend on inner layers, never the reverse.

```
┌─────────────────────────────────────────────────────────────┐
│                     cmd/api/main.go                         │
│              (Composition Root / Wiring)                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              internal/adapter/                        │  │
│  │    HTTP Handlers (Controllers / Adapters)             │  │
│  │    Translates HTTP ↔ UseCase Input/Output             │  │
│  └───────────────────┬───────────────────────────────────┘  │
│                      │ depends on                           │
│  ┌───────────────────▼───────────────────────────────────┐  │
│  │              internal/usecase/                        │  │
│  │    Business Logic (Use Cases / Services)              │  │
│  │    Defines Repository & Provider interfaces           │  │
│  │    Operates on Domain models                          │  │
│  └───────────────────┬───────────────────────────────────┘  │
│                      │ depends on                           │
│  ┌───────────────────▼───────────────────────────────────┐  │
│  │              internal/domain/                         │  │
│  │    Entities, Value Objects, Domain Errors             │  │
│  │    ZERO external dependencies                         │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              infrastructure/                          │  │
│  │    PostgreSQL repos, Redis cache, External APIs       │  │
│  │    Implements interfaces defined in usecase layer     │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Project Directory Structure

```
keyloop-test/
├── cmd/
│   └── api/
│       └── main.go                          # Composition root: wires all layers
│
├── infrastructure/                          # Layer 4: Infrastructure (outermost)
│   ├── postgresql/
│   │   ├── postgre-repository/              #   Implements internal/repository interfaces
│   │   │   └── repositories.go              #   Repository implementations
│   │   └── sqlc/                            #   Generated code by sqlc
│   │       ├── db.go
│   │       ├── models.go
│   │       ├── querier.go
│   │       └── queries.sql.go
│   └── redis/
│       └── cache.go                         #   Redis cache implementation
│
├── internal/
│   ├── adapter/                             # Layer 3: Adapters (HTTP handlers)
│   │   └── http/
│   │
│   ├── domain/                              # Layer 1: Domain (innermost)
│   │   ├── appoinment.go                    #   Appointment entity
│   │   ├── customer.go                      #   Customer entity
│   │   ├── dealership.go                    #   Dealership entity
│   │   ├── errors.go                        #   Domain sentinel errors
│   │   ├── reservation.go                   #   Reservation entity
│   │   ├── service-bay.go                   #   ServiceBay entity
│   │   ├── service.go                       #   Service entity
│   │   ├── technician.go                    #   Technician entity
│   │   ├── technician_skill.go              #   TechnicianSkill entity
│   │   └── vehicle.go                       #   Vehicle entity
│   │
│   ├── repository/                          # Repository Interfaces
│   │   └── interfaces.go                    #   All repository interfaces
│   │
│   └── usecase/                             # Layer 2: Use Cases (business logic)
│       └── book-appointment.go              #   Appointment booking usecase
│
├── migration/                               # Database migrations
│   ├── 000001_init_schema.down.sql
│   └── 000001_init_schema.up.sql
│
├── sqlc/                                    # SQL query definitions for sqlc
│   ├── queries.sql                          #   Combined CRUD queries
│   └── sqlc.yaml                            #   sqlc configuration
│
├── go.mod
├── go.sum
└── Dockerfile
```

---

## Layer 1: Domain (`internal/domain/`)

The **innermost layer**. Contains entities, value objects, typed enums, and domain errors. Has **ZERO imports** from any other layer or external library (except stdlib).

### Rules
- One file per entity/aggregate root.
- Use typed string constants for enums.
- Define sentinel errors here for domain-level error conditions.
- Domain models are pure Go structs — no ORM tags, no JSON tags (those belong in DTOs).
- Domain models CAN have JSON tags when they are also used as API representations, but prefer separate DTOs for complex cases.

### Example: `internal/domain/appointment.go`

```go
package domain

import "time"

// AppointmentStatus represents the lifecycle status of an appointment.
type AppointmentStatus string

const (
    AppointmentStatusPending   AppointmentStatus = "pending"
    AppointmentStatusConfirmed AppointmentStatus = "confirmed"
    AppointmentStatusCancelled AppointmentStatus = "cancelled"
    AppointmentStatusCompleted AppointmentStatus = "completed"
)

// ServiceType represents the type of service to be performed.
type ServiceType string

const (
    ServiceOilChange       ServiceType = "oil_change"
    ServiceTireRotation    ServiceType = "tire_rotation"
    ServiceBrakeInspection ServiceType = "brake_inspection"
    ServiceFullService     ServiceType = "full_service"
)

// Appointment is the core domain entity for a scheduled service.
type Appointment struct {
    ID            string
    CustomerID    string
    VehicleID     string
    DealershipID  string
    ServiceBayID  string
    TechnicianID  string
    ServiceType   ServiceType
    Status        AppointmentStatus
    StartTime     time.Time
    EndTime       time.Time
    Notes         string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### Example: `internal/domain/errors.go`

```go
package domain

import "errors"

var (
    ErrAppointmentNotFound   = errors.New("appointment not found")
    ErrNoAvailableBay        = errors.New("no available service bay for the requested time")
    ErrNoAvailableTechnician = errors.New("no qualified technician available")
    ErrTimeSlotConflict      = errors.New("time slot conflicts with existing appointment")
    ErrInvalidTimeRange      = errors.New("end time must be after start time")
    ErrPastAppointmentTime   = errors.New("cannot book appointment in the past")
)
```

---

## Layer 2: Use Cases (`internal/usecase/<feature>/`)

The **business logic layer**. Orchestrates domain operations, defines **repository interfaces** (ports) that infrastructure must implement.

### Rules
- Each feature/aggregate gets its own sub-package: `internal/usecase/appointment/`.
- **Interfaces are defined HERE** (consumer-side), NOT in infrastructure.
- The usecase struct holds injected dependencies as unexported fields.
- Constructor file: `<entity>.usecase.go` — contains the struct and `New*` function.
- Method files: `service.<operation>.go` — one file per business operation.
- Input/Output structs: define `<Operation>Input` and `<Operation>Output` next to the method that uses them, in the same `service.<operation>.go` file.

### File Naming Convention

| File | Purpose |
|------|---------|
| `<entity>.usecase.go` | Struct definition + constructor (`New*`) |
| `repository.go` | Repository interface definitions (ports) |
| `provider.go` | External service provider interfaces (ports) |
| `service.<operation>.go` | Individual business operation method + its Input/Output types |
| `<entity>.utils.go` | Package-private helper functions |

### Example: `internal/usecase/appointment/repository.go`

```go
package appointmentusecase

import (
    "context"
    "time"

    "keyloop-test/internal/domain"
)

// AppointmentRepository defines persistence operations for appointments.
type AppointmentRepository interface {
    Insert(ctx context.Context, appointment *domain.Appointment) (*domain.Appointment, error)
    FindByID(ctx context.Context, id string) (*domain.Appointment, error)
    FindByDealershipAndTimeRange(ctx context.Context, dealershipID string, start, end time.Time) ([]*domain.Appointment, error)
    Update(ctx context.Context, appointment *domain.Appointment) error
    Delete(ctx context.Context, id string) error
}

// ServiceBayRepository defines persistence operations for service bays.
type ServiceBayRepository interface {
    FindAvailableByDealership(ctx context.Context, dealershipID string, start, end time.Time) ([]*domain.ServiceBay, error)
    FindByID(ctx context.Context, id string) (*domain.ServiceBay, error)
}

// TechnicianRepository defines persistence operations for technicians.
type TechnicianRepository interface {
    FindAvailableByDealershipAndSkill(ctx context.Context, dealershipID string, serviceType domain.ServiceType, start, end time.Time) ([]*domain.Technician, error)
    FindByID(ctx context.Context, id string) (*domain.Technician, error)
}
```

### Example: `internal/usecase/appointment/appointment.usecase.go`

```go
package appointmentusecase

// AppointmentUseCase orchestrates appointment business logic.
type AppointmentUseCase struct {
    appointmentRepo AppointmentRepository
    bayRepo         ServiceBayRepository
    technicianRepo  TechnicianRepository
}

// NewAppointmentUseCase creates a new AppointmentUseCase with injected dependencies.
func NewAppointmentUseCase(
    appointmentRepo AppointmentRepository,
    bayRepo ServiceBayRepository,
    technicianRepo TechnicianRepository,
) *AppointmentUseCase {
    return &AppointmentUseCase{
        appointmentRepo: appointmentRepo,
        bayRepo:         bayRepo,
        technicianRepo:  technicianRepo,
    }
}
```

### Example: `internal/usecase/appointment/service.create.go`

```go
package appointmentusecase

import (
    "context"
    "fmt"
    "time"

    "keyloop-test/internal/domain"
)

// CreateAppointmentInput holds the data needed to create an appointment.
type CreateAppointmentInput struct {
    CustomerID   string
    VehicleID    string
    DealershipID string
    ServiceType  domain.ServiceType
    StartTime    time.Time
    Duration     time.Duration
}

// CreateAppointmentOutput holds the result of a successful appointment creation.
type CreateAppointmentOutput struct {
    AppointmentID string
    ServiceBayID  string
    TechnicianID  string
    StartTime     time.Time
    EndTime       time.Time
}

// CreateAppointment validates availability and creates a confirmed appointment.
func (uc *AppointmentUseCase) CreateAppointment(ctx context.Context, input CreateAppointmentInput) (*CreateAppointmentOutput, error) {
    endTime := input.StartTime.Add(input.Duration)

    // Validate time range
    if endTime.Before(input.StartTime) || endTime.Equal(input.StartTime) {
        return nil, domain.ErrInvalidTimeRange
    }
    if input.StartTime.Before(time.Now()) {
        return nil, domain.ErrPastAppointmentTime
    }

    // Check service bay availability
    availableBays, err := uc.bayRepo.FindAvailableByDealership(ctx, input.DealershipID, input.StartTime, endTime)
    if err != nil {
        return nil, fmt.Errorf("failed to check bay availability: %w", err)
    }
    if len(availableBays) == 0 {
        return nil, domain.ErrNoAvailableBay
    }

    // Check technician availability
    availableTechnicians, err := uc.technicianRepo.FindAvailableByDealershipAndSkill(ctx, input.DealershipID, input.ServiceType, input.StartTime, endTime)
    if err != nil {
        return nil, fmt.Errorf("failed to check technician availability: %w", err)
    }
    if len(availableTechnicians) == 0 {
        return nil, domain.ErrNoAvailableTechnician
    }

    // Select first available bay and technician
    selectedBay := availableBays[0]
    selectedTechnician := availableTechnicians[0]

    // Create the appointment
    appointment := &domain.Appointment{
        CustomerID:   input.CustomerID,
        VehicleID:    input.VehicleID,
        DealershipID: input.DealershipID,
        ServiceBayID: selectedBay.ID,
        TechnicianID: selectedTechnician.ID,
        ServiceType:  input.ServiceType,
        Status:       domain.AppointmentStatusConfirmed,
        StartTime:    input.StartTime,
        EndTime:      endTime,
    }

    created, err := uc.appointmentRepo.Insert(ctx, appointment)
    if err != nil {
        return nil, fmt.Errorf("failed to persist appointment: %w", err)
    }

    return &CreateAppointmentOutput{
        AppointmentID: created.ID,
        ServiceBayID:  selectedBay.ID,
        TechnicianID:  selectedTechnician.ID,
        StartTime:     created.StartTime,
        EndTime:       created.EndTime,
    }, nil
}
```

---

## Layer 3: Adapters (`internal/adapter/`)

The **interface adapter layer**. Translates between external interfaces (HTTP, gRPC, CLI) and the use case layer.

### Rules
- Grouped by transport protocol: `internal/adapter/http/<feature>/`.
- Handler struct holds a reference to the **use case**, never directly to a repository.
- Request/Response DTOs are defined in `dto.go` — they are NOT domain models.
- Validation of HTTP-specific concerns (required fields, format) happens here.
- Business validation happens in the use case layer.
- Constructor file: `handler.go` — struct + `New*` function.
- Handler files: `handler.<operation>.go` — one handler per file.

### File Naming Convention

| File | Purpose |
|------|---------|
| `handler.go` | Handler struct + constructor |
| `handler.<operation>.go` | Individual HTTP handler method |
| `dto.go` | Request/Response DTOs with JSON tags |
| `routes.go` | Route registration helper |

### Example: `internal/adapter/http/appointment/handler.go`

```go
package appointmenthandler

import appointmentusecase "keyloop-test/internal/usecase/appointment"

// AppointmentHandler handles HTTP requests for appointment operations.
type AppointmentHandler struct {
    appointmentUseCase *appointmentusecase.AppointmentUseCase
}

// NewAppointmentHandler creates a new AppointmentHandler.
func NewAppointmentHandler(uc *appointmentusecase.AppointmentUseCase) *AppointmentHandler {
    return &AppointmentHandler{appointmentUseCase: uc}
}
```

### Example: `internal/adapter/http/appointment/dto.go`

```go
package appointmenthandler

import "time"

// CreateAppointmentRequest is the HTTP request body for creating an appointment.
type CreateAppointmentRequest struct {
    CustomerID   string `json:"customer_id"`
    VehicleID    string `json:"vehicle_id"`
    DealershipID string `json:"dealership_id"`
    ServiceType  string `json:"service_type"`
    StartTime    string `json:"start_time"`
    DurationMins int    `json:"duration_mins"`
}

// CreateAppointmentResponse is the HTTP response body after creating an appointment.
type CreateAppointmentResponse struct {
    AppointmentID string    `json:"appointment_id"`
    ServiceBayID  string    `json:"service_bay_id"`
    TechnicianID  string    `json:"technician_id"`
    StartTime     time.Time `json:"start_time"`
    EndTime       time.Time `json:"end_time"`
    Status        string    `json:"status"`
}

// ErrorResponse is the standard error response body.
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}
```

### Example: `internal/adapter/http/appointment/handler.create.go`

```go
package appointmenthandler

import (
    "encoding/json"
    "errors"
    "net/http"
    "time"

    "keyloop-test/internal/domain"
    appointmentusecase "keyloop-test/internal/usecase/appointment"
)

// CreateAppointment handles POST /appointments.
func (h *AppointmentHandler) CreateAppointment(w http.ResponseWriter, r *http.Request) {
    var req CreateAppointmentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: "Invalid JSON body"})
        return
    }

    // Parse and validate request
    startTime, err := time.Parse(time.RFC3339, req.StartTime)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_time", Message: "start_time must be RFC3339 format"})
        return
    }

    input := appointmentusecase.CreateAppointmentInput{
        CustomerID:   req.CustomerID,
        VehicleID:    req.VehicleID,
        DealershipID: req.DealershipID,
        ServiceType:  domain.ServiceType(req.ServiceType),
        StartTime:    startTime,
        Duration:     time.Duration(req.DurationMins) * time.Minute,
    }

    output, err := h.appointmentUseCase.CreateAppointment(r.Context(), input)
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrNoAvailableBay):
            writeJSON(w, http.StatusConflict, ErrorResponse{Error: "no_bay", Message: err.Error()})
        case errors.Is(err, domain.ErrNoAvailableTechnician):
            writeJSON(w, http.StatusConflict, ErrorResponse{Error: "no_technician", Message: err.Error()})
        case errors.Is(err, domain.ErrInvalidTimeRange), errors.Is(err, domain.ErrPastAppointmentTime):
            writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_time", Message: err.Error()})
        default:
            writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal", Message: "Something went wrong"})
        }
        return
    }

    writeJSON(w, http.StatusCreated, CreateAppointmentResponse{
        AppointmentID: output.AppointmentID,
        ServiceBayID:  output.ServiceBayID,
        TechnicianID:  output.TechnicianID,
        StartTime:     output.StartTime,
        EndTime:       output.EndTime,
        Status:        string(domain.AppointmentStatusConfirmed),
    })
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}
```

---

## Layer 4: Infrastructure (`infrastructure/`)

The **outermost layer**. Contains concrete implementations of interfaces (ports) defined in the use case layer.

### Rules
- Grouped by technology: `infrastructure/postgresql/`, `infrastructure/redis/`.
- Each feature's repository implementation gets a sub-package: `infrastructure/postgresql/appointment/`.
- Implements interfaces from `internal/usecase/<feature>/repository.go`.
- Contains database-specific concerns: SQL queries, ORM mappings, connection management.
- Use **sqlc** for type-safe SQL query generation where applicable.
- Mapper functions (`toModel` / `toDomain`) to convert between DB models and domain entities.

### File Naming Convention

| File | Purpose |
|------|---------|
| `connection.go` | Database connection setup (`infrastructure/postgresql/`) |
| `<entity>.repository.go` | Repository interface implementation |

### Example: `infrastructure/postgresql/connection.go`

```go
package postgresql

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
)

// InitPostgresClient creates and returns a PostgreSQL connection pool.
func InitPostgresClient() *pgxpool.Pool {
    connStr := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
        os.Getenv("POSTGRES_USER"),
        os.Getenv("POSTGRES_PASSWORD"),
        os.Getenv("POSTGRES_HOST"),
        os.Getenv("POSTGRES_PORT"),
        os.Getenv("POSTGRES_DB"),
    )

    pool, err := pgxpool.New(context.Background(), connStr)
    if err != nil {
        log.Fatalf("Unable to connect to PostgreSQL: %v", err)
    }

    if err := pool.Ping(context.Background()); err != nil {
        log.Fatalf("PostgreSQL ping failed: %v", err)
    }

    log.Println("Connected to PostgreSQL successfully")
    return pool
}
```

### Example: `infrastructure/postgresql/appointment/appointment.repository.go`

```go
package appointmentpostgre

import (
    "context"
    "time"

    "keyloop-test/internal/domain"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
)

// PostgreAppointmentRepository implements AppointmentRepository using PostgreSQL.
type PostgreAppointmentRepository struct {
    pool *pgxpool.Pool
}

// NewPostgreAppointmentRepository creates a new PostgreSQL-backed appointment repository.
func NewPostgreAppointmentRepository(pool *pgxpool.Pool) *PostgreAppointmentRepository {
    return &PostgreAppointmentRepository{pool: pool}
}

func (repo *PostgreAppointmentRepository) Insert(ctx context.Context, appointment *domain.Appointment) (*domain.Appointment, error) {
    id := uuid.New().String()
    now := time.Now()

    _, err := repo.pool.Exec(ctx,
        `INSERT INTO appointments (id, customer_id, vehicle_id, dealership_id, service_bay_id, technician_id, service_type, status, start_time, end_time, notes, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
        id, appointment.CustomerID, appointment.VehicleID, appointment.DealershipID,
        appointment.ServiceBayID, appointment.TechnicianID, appointment.ServiceType,
        appointment.Status, appointment.StartTime, appointment.EndTime, appointment.Notes, now, now,
    )
    if err != nil {
        return nil, err
    }

    appointment.ID = id
    appointment.CreatedAt = now
    appointment.UpdatedAt = now
    return appointment, nil
}

// ... other interface methods follow the same pattern
```

---

## Composition Root (`cmd/api/main.go`)

The composition root wires all layers together. This is the ONLY place where concrete implementations are instantiated and injected into use cases.

### Wiring Order
1. **Infrastructure**: Database connections, cache clients, external API clients
2. **Repositories**: Concrete repository implementations (pass DB pool)
3. **Use Cases**: Business logic (inject repository interfaces)
4. **Handlers**: HTTP handlers (inject use cases)
5. **Router**: Register routes and start server

### Example: `cmd/api/main.go`

```go
package main

import (
    "log"
    "net/http"

    postgresql "keyloop-test/infrastructure/postgresql"
    appointmentpostgre "keyloop-test/infrastructure/postgresql/appointment"
    appointmentusecase "keyloop-test/internal/usecase/appointment"
    appointmenthandler "keyloop-test/internal/adapter/http/appointment"
)

func main() {
    // 1. Infrastructure
    pool := postgresql.InitPostgresClient()
    defer pool.Close()

    // 2. Repositories
    appointmentRepo := appointmentpostgre.NewPostgreAppointmentRepository(pool)
    bayRepo := appointmentpostgre.NewPostgreServiceBayRepository(pool)
    technicianRepo := appointmentpostgre.NewPostgreTechnicianRepository(pool)

    // 3. Use Cases
    appointmentUC := appointmentusecase.NewAppointmentUseCase(appointmentRepo, bayRepo, technicianRepo)

    // 4. Handlers
    appointmentH := appointmenthandler.NewAppointmentHandler(appointmentUC)

    // 5. Router
    mux := http.NewServeMux()
    mux.HandleFunc("POST /appointments", appointmentH.CreateAppointment)
    mux.HandleFunc("GET /appointments/{id}", appointmentH.GetAppointment)

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

---

## Dependency Rule — Visual Summary

```
ALLOWED                               FORBIDDEN
─────────────────────────             ─────────────────────────
adapter → usecase → domain           domain → usecase
adapter → domain                     domain → adapter
infrastructure → usecase             domain → infrastructure
infrastructure → domain              usecase → adapter
cmd → ALL layers                      usecase → infrastructure (concrete)
```

The **usecase layer** defines interfaces (ports). The **infrastructure layer** implements them. The **adapter layer** calls use case methods. The **domain** depends on nothing.

---

## Checklist for Adding a New Feature

1. **Domain**: Create entity struct + enums + sentinel errors in `internal/domain/`.
2. **Use Case**: Create `internal/usecase/<feature>/` with:
   - `repository.go` — define repository interface
   - `<feature>.usecase.go` — struct + constructor
   - `service.<operation>.go` — each business method + Input/Output types
3. **Infrastructure**: Create `infrastructure/postgresql/<feature>/` with:
   - `<feature>.repository.go` — implements the repository interface
4. **Adapter**: Create `internal/adapter/http/<feature>/` with:
   - `handler.go` — handler struct + constructor
   - `dto.go` — request/response DTOs
   - `handler.<operation>.go` — HTTP handler methods
5. **Wire**: Update `cmd/api/main.go` — instantiate repo → usecase → handler → register routes.
6. **SQL**: Add migration in `sqlc/` if new tables are needed.
