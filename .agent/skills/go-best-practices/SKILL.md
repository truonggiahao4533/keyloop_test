# Go Best Practices Skill

## When to Use This Skill

Apply these conventions whenever writing, reviewing, or modifying Go code in this repository. This is the authoritative reference for code style, naming, error handling, concurrency, and project conventions.

---

## 1. Naming Conventions

### Packages

- Use **short, lowercase, single-word** package names. No underscores, no camelCase.
- Package name should describe what it *provides*, not what it *contains*.
- Avoid generic names like `util`, `common`, `helpers` — prefer domain-specific names.

```go
// GOOD
package appointment
package technician

// BAD
package appointmentService
package helpers
package utils
```

### When a package groups related concerns, suffix with the layer role:

```go
package appointmentusecase     // usecase layer
package appointmentrepository  // repository interface layer
package appointmentcontroller  // controller/handler layer
```

### Variables and Functions

- Use **camelCase** for unexported, **PascalCase** for exported.
- Receiver names: short (1–2 chars), consistent across methods of the same type.
- Avoid stuttering: `appointment.NewAppointment()` → prefer `appointment.New()`.
- Boolean variables/fields should read as assertions: `IsActive`, `HasConflict`, `CanBook`.

```go
// GOOD
func (uc *UseCase) CreateAppointment(ctx context.Context, input CreateInput) (*Appointment, error)

// BAD
func (useCase *UseCase) CreateNewAppointment(ctx context.Context, input CreateInput) (*Appointment, error)
```

### Constants and Enums

- Use typed string constants for enums, grouped with `const` blocks.
- Type name is singular (`ServiceType`, not `ServiceTypes`).

```go
type ServiceType string

const (
    ServiceOilChange  ServiceType = "oil_change"
    ServiceTireRotation ServiceType = "tire_rotation"
    ServiceBrakeInspection ServiceType = "brake_inspection"
)
```

---

## 2. Error Handling

### Always wrap errors with context using `fmt.Errorf` and `%w`:

```go
func (uc *UseCase) CreateAppointment(ctx context.Context, input CreateInput) (*domain.Appointment, error) {
    bay, err := uc.bayRepo.FindAvailable(ctx, input.DealershipID, input.StartTime, input.EndTime)
    if err != nil {
        return nil, fmt.Errorf("failed to find available service bay: %w", err)
    }
    // ...
}
```

### Define domain-specific sentinel errors in the domain layer:

```go
package domain

import "errors"

var (
    ErrAppointmentNotFound  = errors.New("appointment not found")
    ErrNoAvailableBay       = errors.New("no available service bay for the requested time")
    ErrNoAvailableTechnician = errors.New("no qualified technician available for the requested time")
    ErrTimeSlotConflict     = errors.New("requested time slot conflicts with an existing appointment")
)
```

### Check sentinel errors with `errors.Is()`:

```go
if errors.Is(err, domain.ErrAppointmentNotFound) {
    // handle not found
}
```

### Never ignore errors. If intentionally discarding, document why:

```go
_ = logger.Sync() // best-effort flush; error is non-critical at shutdown
```

---

## 3. Context Usage

- **First parameter** of any function that does I/O, database calls, or crosses service boundaries must be `context.Context`.
- Never store `context.Context` in a struct.
- Use `context.Background()` only at the top level (main, test setup). Within business logic, always propagate the received `ctx`.

```go
// GOOD
func (uc *UseCase) GetAppointment(ctx context.Context, id string) (*domain.Appointment, error)

// BAD — missing context
func (uc *UseCase) GetAppointment(id string) (*domain.Appointment, error)
```

---

## 4. Struct Design

### Use constructor functions (New*) for initialization:

```go
type AppointmentUseCase struct {
    appointmentRepo repository.AppointmentRepository
    bayRepo         repository.ServiceBayRepository
    technicianRepo  repository.TechnicianRepository
}

func NewAppointmentUseCase(
    appointmentRepo repository.AppointmentRepository,
    bayRepo repository.ServiceBayRepository,
    technicianRepo repository.TechnicianRepository,
) *AppointmentUseCase {
    return &AppointmentUseCase{
        appointmentRepo: appointmentRepo,
        bayRepo:         bayRepo,
        technicianRepo:  technicianRepo,
    }
}
```

### Keep fields unexported unless there is a strong reason to export them.

### Use value receivers for small, read-only types; pointer receivers for mutating or large types:

```go
// Pointer receiver — mutates or is large
func (a *Appointment) Cancel() { a.Status = StatusCancelled }

// Value receiver — small, read-only
func (s ServiceType) String() string { return string(s) }
```

---

## 5. Interface Design

- Define interfaces **where they are consumed**, not where they are implemented (consumer-side interfaces).
- Keep interfaces small — prefer 3–5 methods max. Split larger contracts.
- Name interfaces as verbs/nouns describing behavior, e.g., `Repository`, `Provider`, `Notifier`.

```go
// Defined in the usecase layer (consumer), NOT in the infrastructure layer (implementor)
package repository

type AppointmentRepository interface {
    Insert(ctx context.Context, appointment *domain.Appointment) (*domain.Appointment, error)
    FindByID(ctx context.Context, id string) (*domain.Appointment, error)
    FindByDealershipAndTimeRange(ctx context.Context, dealershipID string, start, end time.Time) ([]*domain.Appointment, error)
    Update(ctx context.Context, appointment *domain.Appointment) error
    Delete(ctx context.Context, id string) error
}
```

---

## 6. Concurrency

- Use goroutines sparingly. Prefer synchronous code unless concurrency provides measurable benefit.
- Always manage goroutine lifecycle — use `context.Context` cancellation, `sync.WaitGroup`, or channels for coordination.
- For fire-and-forget operations (e.g., notifications), use goroutines but log errors:

```go
go func(sub *domain.Subscription) {
    notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := uc.notifier.Send(notifyCtx, sub, payload); err != nil {
        log.Printf("notification failed for subscription %s: %v", sub.ID, err)
    }
}(subscription)
```

---

## 7. Testing Conventions

- Table-driven tests for functions with multiple input/output scenarios.
- Use `testify/assert` or `testify/require` for assertions.
- Mock interfaces at the boundary — use generated mocks or hand-written fakes.
- Test file naming: `<filename>_test.go` in the same package.

```go
func TestCreateAppointment(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateInput
        wantErr error
    }{
        {
            name:    "success",
            input:   CreateInput{VehicleID: "v1", ServiceType: domain.ServiceOilChange},
            wantErr: nil,
        },
        {
            name:    "no available bay",
            input:   CreateInput{VehicleID: "v1", ServiceType: domain.ServiceOilChange},
            wantErr: domain.ErrNoAvailableBay,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // arrange, act, assert
        })
    }
}
```

---

## 8. Logging

- Use structured logging (`log/slog` for Go 1.21+, or `log.Printf` as a baseline).
- Log at service boundaries and on errors. Avoid logging inside domain/pure-logic code.
- Include contextual fields: request ID, entity IDs, operation name.

---

## 9. JSON Serialization

- Use `json:"snake_case"` tags on all exported fields.
- Use `omitempty` for optional/nullable fields.
- Use typed constants (not raw strings) for enum-like JSON values.

```go
type Appointment struct {
    ID          string          `json:"id"`
    VehicleID   string          `json:"vehicle_id"`
    ServiceType ServiceType     `json:"service_type"`
    Status      AppointmentStatus `json:"status"`
    StartTime   time.Time       `json:"start_time"`
    EndTime     time.Time       `json:"end_time"`
    CreatedAt   time.Time       `json:"created_at"`
    Metadata    map[string]any  `json:"metadata,omitempty"`
}
```

---

## 10. File Organization Within a Package

- **One primary type per file** when the type is large (e.g., a usecase struct with many methods).
- Split by concern: `<entity>.usecase.go` (constructor + struct), `service.<operation>.go` (individual methods).
- Utility/helper functions that are specific to the package go in `<package>.utils.go`.

```
internal/usecase/appointment/
├── appointment.usecase.go          # struct + constructor
├── service.create-appointment.go   # CreateAppointment method
├── service.check-availability.go   # CheckAvailability method
└── appointment.utils.go            # package-level helpers
```

---

## 11. Dependency Injection

- All dependencies are injected through constructor functions (`New*`).
- Never use global state or package-level variables for dependencies.
- Wire everything in `main.go` or a dedicated composition root (`cmd/api/main.go`).
- Order of initialization: infrastructure → repositories → usecases → controllers → server.

---

## 12. Module and Import Conventions

- Group imports in 3 blocks: stdlib, project-internal, third-party.
- Use descriptive import aliases when package names would collide:

```go
import (
    "context"
    "fmt"

    appointmentusecase "keyloop-test/internal/usecase/appointment"
    appointmentrepo "keyloop-test/infrastructure/postgresql/appointment"

    "github.com/jackc/pgx/v5/pgxpool"
)
```
