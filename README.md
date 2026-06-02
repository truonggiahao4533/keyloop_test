# Keyloop Appointment Booking API

A RESTful API for booking vehicle service appointments at dealerships. Built with Go, Gin, PostgreSQL, Redis, and OpenTelemetry.

## Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Environment Variables](#environment-variables)
- [Build](#build)
- [Run](#run)
- [Migrations](#migrations)
- [API Reference](#api-reference)
- [Testing](#testing)
- [Observability](#observability)
- [AI Collaboration Narrative](#ai-collaboration-narrative)

---

## Architecture

The project follows **Clean Architecture** with strict layer separation:

```
cmd/api/           – Entrypoint, dependency wiring
internal/
  domain/          – Core entities and business rules (no external deps)
  repository/      – Repository interfaces (contracts only)
  usecase/         – Application logic, orchestrates domain + repos
  adapter/http/    – Gin handlers and router (HTTP concerns only)
infrastructure/
  postgresql/      – sqlc-generated queries + repository implementations
  redis/           – Service metadata cache
  telemetry/       – OpenTelemetry SDK setup
migration/         – SQL migration files (golang-migrate)
sqlc/              – SQL query definitions and sqlc config
test/              – Unit, usecase, and integration tests
```

The dependency rule flows inward: `adapter → usecase → domain`. Infrastructure implements the repository interfaces defined in `internal/repository/`.

**Conflict prevention** is enforced at the database level using PostgreSQL exclusion constraints (`EXCLUDE USING gist`) on `tstzrange(start_time, end_time)` for both service bays and technicians — the database is the source of truth for double-booking prevention.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Docker + Compose | 24+ | Run all services |
| Go | 1.25+ | Local dev / tests |
| `golang-migrate` | latest | Running migrations locally |
| `swag` CLI | v1.x | Regenerating Swagger docs |

Install `golang-migrate`:
```bash
make install-tools
```

---

## Environment Variables

Copy and fill in `.env` at the project root:

```dotenv
PORT=8081

POSTGRES_HOST=postgresql
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=keyloop

REDIS_ADDR=redis:6379
```

> When running locally (outside Docker), set `POSTGRES_HOST=localhost` and `REDIS_ADDR=localhost:6379`, and note the host-mapped ports (`5433` for Postgres, `6380` for Redis).

---

## Build

**Docker image:**
```bash
make build
```

This builds the `keyloop-test` image using the multi-stage `Dockerfile`.

---

## Run

**Start all services (app + PostgreSQL + Redis + auto-migration):**
```bash
make up
```

Docker Compose starts services in dependency order:
1. PostgreSQL (with health check)
2. Redis (with health check)
3. `migrate` container — applies all pending migrations, then exits
4. Application container — waits for migration container to succeed

**Tail application logs:**
```bash
make watch
```

**Stop all services:**
```bash
make down
```

**Local dev (outside Docker):**

Ensure PostgreSQL and Redis are reachable on their host-mapped ports, then:
```bash
go run ./cmd/api/...
```

The server defaults to port `8080` when `PORT` is unset.

---

## Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate) and SQL files in `migration/`.

| Command | Description |
|---------|-------------|
| `make migrate-up` | Apply all pending migrations (runs via Docker, uses the compose network) |
| `make migrate-down` | Roll back the last migration |
| `make migrate-down-all` | Roll back all migrations (destructive) |
| `make migrate-version` | Print current migration version |
| `make migrate-force V=<n>` | Force-set version without running SQL (use after manual fix) |

> `make up` runs migrations automatically via the `migrate` service in Compose. Use the Makefile targets above only for local or manual management.

**Regenerate Swagger docs** (after changing handler annotations):
```bash
make swagger
```

---

## API Reference

The API is documented with Swagger UI, available at:

```
http://localhost:8081/swagger/index.html
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/appointments` | Book a new appointment |
| `GET` | `/api/v1/appointments` | List appointments (filter by customer or dealership) |
| `GET` | `/api/v1/appointments/available-slots` | List available 30-minute slots within 7 days |

### Book an Appointment

```http
POST /api/v1/appointments
Content-Type: application/json

{
  "customer_id":        "c1a2b3c4-d5e6-7890-abcd-ef1234567890",
  "dealership_id":      "d1e2f3a4-b5c6-7890-abcd-ef1234567890",
  "vehicle_id":         "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "services":           ["<service-uuid>"],
  "desired_start_time": "2026-07-01T09:00:00Z"
}
```

**Response 201:**
```json
{
  "appointment_id":   "f1a2b3c4-...",
  "duration_minutes": 75,
  "message":          "Appointment booked successfully"
}
```

### Get Available Slots

```http
GET /api/v1/appointments/available-slots
  ?dealership_id=<uuid>
  &vehicle_id=<uuid>
  &customer_id=<uuid>
  &services=<uuid>
  &desired_time=2026-07-01T00:00:00Z
```

Returns 30-minute slots within the dealership's working hours over the next 7 days where a qualified technician and a service bay are both free.

### List Appointments

```http
GET /api/v1/appointments?customer_id=<uuid>
GET /api/v1/appointments?dealership_id=<uuid>
GET /api/v1/appointments?customer_id=<uuid>&dealership_id=<uuid>
```

At least one filter is required.

### Error Responses

| Status | Condition |
|--------|-----------|
| `400` | Missing required fields |
| `404` | Dealership or vehicle not found |
| `409` | Time slot conflict (bay or technician already booked) |
| `422` | Slot outside working hours, not a working day, or date in the past |
| `500` | Internal error |

---

## Testing

The test suite is split across three build tags:

```bash
# Domain entity unit tests
make test-domain

# Usecase logic tests (mocked repositories)
make test-usecase

# Integration tests (real Postgres via Testcontainers)
make test-integration
```

**Domain tests** verify business rules in isolation — working-day checks, slot boundary conditions, appointment duration calculations — with no external dependencies.

**Usecase tests** exercise the booking and availability logic using mock repository implementations, verifying error propagation and domain-rule enforcement.

**Integration tests** spin up a real PostgreSQL instance via [Testcontainers](https://golang.testcontainers.org/), apply migrations, and exercise the full repository layer against actual SQL — including the exclusion-constraint conflict detection.

---

## Observability

The application is instrumented with **OpenTelemetry** across three layers:

| Layer | Instrumentation |
|-------|----------------|
| HTTP | `otelgin` middleware creates a root span per request |
| Usecase | Manual child spans per function (`BookAppointment`, `AvailableSlots`, `ListAppointments`) |
| Repository calls | Grandchild spans wrap each DB call (`repo.GetDealership`, `repo.CreateAppointment`, etc.) |

**Current exporter:** stdout (`stdouttrace` with pretty-print). Completed spans are printed as JSON to the application's stdout after each request. View them with:

```bash
make watch | jq .
```

**Switching to a visual UI (Jaeger):**

1. Start Jaeger:
   ```bash
   docker run -d -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one
   ```

2. In [infrastructure/telemetry/otel.go](infrastructure/telemetry/otel.go), replace the stdout exporter with OTLP:
   ```go
   // swap these two imports:
   // "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
   "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

   traceExporter, err := otlptracegrpc.New(ctx,
       otlptracegrpc.WithEndpoint("localhost:4317"),
       otlptracegrpc.WithInsecure(),
   )
   ```

3. Open `http://localhost:16686` to browse traces.

---

## AI Collaboration Narrative

This project was built in collaboration with **Claude Code** (Anthropic's AI coding assistant). The following describes the strategy used, how output was verified, and how quality was maintained throughout.

### Strategy for Guiding the AI

The collaboration treated Claude as a **capable senior pair-programmer**, not an autonomous agent. Each task was framed as a specific, bounded engineering problem with clear inputs, constraints, and expected outputs — never vague delegation.

**Prompt discipline mattered most.** Prompts included:
- The exact file and function to change
- The specific invariant or bug to fix
- The reason *why* (so the AI could make judgment calls at the edges)
- Explicit constraints ("don't touch the repository layer", "keep the named return `err` for the defer closure")

For example, when adding repo-level spans, the instruction specified the exact pattern — start the span before the call, record error on the span before ending it, then end the span before acting on the error — rather than asking Claude to "add observability". This level of specificity prevented the AI from introducing its own design decisions that would need to be unwound.

**Iterative scope control** was also key. Each session focused on one architectural concern at a time (e.g., the OTel middleware, then the usecase spans, then the cross-midnight bug). Mixing concerns in a single prompt increases the surface area for error and makes verification harder.

### Verifying and Refining Output

Every AI-generated change was verified through multiple lenses before being accepted:

1. **Compile check first.** The project was built (`go build ./...`) after each change. Unused imports, wrong types, and incorrect function signatures surface immediately — Go's strict compiler is a fast feedback loop that caught several issues (e.g., imports that became unused after switching from `http.Server` to `r.Run()`).

2. **Read the diff, not the description.** Claude's explanation of what it did was treated as a hypothesis to verify, not a fact. The actual file diff was always reviewed against the intent of the original request.

3. **Test execution for logic changes.** Domain-level changes (the `IsSlotWithinHours` midnight fix) were verified by running the domain test suite, not just by reading the code.

4. **Edge-case reasoning.** For the cross-midnight slot bug, the root cause was traced manually: `timeOfDay(2026-06-12T00:00:00Z)` returns `0`, and `0 <= CloseTime` is always true for any valid close time. The fix was verified by working through the invariant — slots where `start.Date() != end.Date()` must be rejected before the time-of-day comparison runs.

5. **Pattern consistency.** When Claude introduced two patterns (`repoErr` for domain-wrapping calls, named return `err` for pass-through calls), each pattern was checked for correctness independently: does `repoErr` let the repo span record the raw DB error while the outer span records the domain error? Does the `defer` closure capture the correct variable?

### Ensuring Final Code Quality

Several practices kept the output from drifting into over-engineered or incorrect territory:

- **No speculative additions.** The AI was explicitly redirected when it proposed adding error handling for impossible cases, wrapping return types in unnecessary structs, or adding comments that simply restated the code.

- **Layer discipline enforcement.** The original instruction for repo-level spans was: "add spans in the usecase layer, not in the repository layer." When Claude's initial draft added code to the repo implementations instead, the constraint was re-stated and the change was rejected.

- **Keeping the defer pattern correct.** The `defer func() { if err != nil { span.RecordError(err) } span.End() }()` closure works because `err` is a named return. Claude correctly identified this; the review step confirmed that none of the early returns inside the function assign to a different variable named `err` that would shadow the named return.

- **Reviewing the full function, not just the changed lines.** When a function received new spans, the entire function body was re-read to check for unreachable `span.End()` calls, spans that could be started but never ended on an early return path, or `rCtx` variables created but not passed to the repo call.

The end result is code that is consistent in style, correct in its invariants, and observable at every layer — without unnecessary abstraction or defensive code for cases that cannot occur.
