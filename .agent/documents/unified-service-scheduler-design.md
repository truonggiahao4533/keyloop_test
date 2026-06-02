# Unified Service Scheduler — System Design Summary

## 1. Overview

A dealership appointment booking system that allows users to book vehicle service appointments by checking real-time availability of service bays and qualified technicians, then confirming a persistent appointment record.

---

## 2. Data Classification

| Data | Change Frequency | Strategy |
|---|---|---|
| Dealership active status | Rarely | Cache (TTL: 1–24 hrs) |
| Services offered | Rarely | Cache (TTL: 1–24 hrs) |
| Technician qualifications | Occasionally | Always DB  |
| Registered vehicles | Occasionally | Cache (TTL: 30 min) |
| Service bay availability | Frequent | Always DB |
| Appointments | Constant | Always DB |

---

## 3. Slot Duration

- **Unit size:** 30 minutes per slot
- **Services span multiple slots** based on their duration

| Service | Duration | Slots |
|---|---|---|
| Oil Change | 30 min | 1 slot |
| Tire Rotation | 30 min | 1 slot |
| Brake Service | 60 min | 2 slots |
| Full Inspection | 90 min | 3 slots |
| Engine Repair | 4 hrs | 8 slots |

---

## 4. Booking Flow

```
User provides:
  - Desired date
  - Dealership
  - Vehicle
  - Service Type
        │
        ▼
[Step 1] Validate inputs
         - Is desired date in the past?     
         - Is dealership valid & active?    ← DB 
         - Does services exist ?            ← Redis Cache
        │
        ▼
[Step 2] Calculate service duration
         e.g. Total Services Duration = 60 min = 1 consecutive slots
        │
        ▼
[Step 3] Query available slots                  ← DB (exclusion-based)
         Find windows where BOTH a bay AND
         a qualified technician are free
         for the full duration
        │
        ▼
[Step 4] Return slot list to user
         e.g. "Mon 9:00, Mon 10:30, Tue 8:00..."
        │
        ▼
[Step 5] User picks a slot 
        │
        ▼
[Step 6] BEGIN TRANSACTION
           Validate slot still available        ← DB check inside tx
           INSERT into appointments             ← EXCLUDE gist prevents double-booking
         COMMIT
         │
         ├── FAIL (conflict) → "Slot just taken, pick another"
         │
         SUCCESS → appointment confirmed
```

> **Note — Reservation Service (future):** A `reservations` table acting as a short-lived lock (5–10 min TTL) should be introduced if a payment or user-confirmation step is required between slot selection and final booking. Without that intermediate step, direct transactional insert is simpler and equally safe due to the `EXCLUDE USING gist` constraints on the appointments table.

---

## 5. Availability Strategy — Exclusion-Based

Rather than storing all free slots, **only busy slots (appointments) are stored**. Free slots are derived at query time.

```
FREE = Business Hours − Appointments Table
```

### Why Exclusion Over Pre-computation

| Factor | Exclusion (chosen) | Pre-compute + Sliding Window |
|---|---|---|
| Storage | Small (busy only) ✅ | Large (all combinations) |
| Booking range | Unlimited ✅ | Bounded by window size |
| Query complexity | Medium | Simple SELECT |
| Maintenance | None needed ✅ | Requires daily cron job |
| Accuracy | Always accurate ✅ | Needs sync on every booking |

### Required Indexes

```sql
CREATE INDEX idx_appointments_bay_time
  ON appointments(bay_id, start_time, end_time);

CREATE INDEX idx_appointments_tech_time
  ON appointments(technician_id, start_time, end_time);

CREATE INDEX idx_appointments_dealership_date
  ON appointments(dealership_id, start_time);

CREATE INDEX idx_tech_qualifications
  ON technician_qualifications(technician_id, service_type_id);
```

### Availability Query Logic (Simplified)

```sql
SELECT slot
FROM generate_30min_slots(:date, '08:00', '18:00') AS slot

WHERE
  -- At least 1 bay is free for the full duration
  EXISTS (
    SELECT 1 FROM service_bays b
    WHERE b.dealership_id = :dealershipId
    AND b.id NOT IN (
      SELECT bay_id FROM appointments
      WHERE tsrange(start_time, end_time) &&
            tsrange(slot, slot + :duration)
    )
  )

  AND

  -- At least 1 qualified technician is free for the full duration
  EXISTS (
    SELECT 1 FROM technicians t
    JOIN technician_qualifications q ON q.tech_id = t.id
    WHERE t.dealership_id = :dealershipId
    AND q.service_type_id = :serviceTypeId
    AND t.id NOT IN (
      SELECT technician_id FROM appointments
      WHERE tsrange(start_time, end_time) &&
            tsrange(slot, slot + :duration)
    )
  )
```

---

## 6. Race Condition Protection — Transactional Insert

### The Problem (TOCTOU Race Condition)

```
User A checks Mon 10:00 → FREE
User B checks Mon 10:00 → FREE
Both click Book simultaneously
  → Both insert appointment
  → DOUBLE BOOKING 💥
```

### The Solution — EXCLUDE USING gist on Appointments

The `appointments` table carries `EXCLUDE USING gist` constraints directly, so a concurrent insert for an overlapping slot will fail with a conflict error rather than producing a double booking:

```sql
-- Prevent overlapping bookings for the same bay
CONSTRAINT no_overlap_bay EXCLUDE USING gist (
  service_bay_id WITH =,
  tstzrange(start_time, end_time) WITH &&
) WHERE (deleted_at IS NULL AND status IN ('pending', 'confirmed')),

-- Prevent overlapping bookings for the same technician
CONSTRAINT no_overlap_tech EXCLUDE USING gist (
  technician_id WITH =,
  tstzrange(start_time, end_time) WITH &&
) WHERE (deleted_at IS NULL AND status IN ('pending', 'confirmed'))
```

One of the two concurrent inserts will win; the other receives a constraint violation which the API surfaces as "slot just taken, pick another."

---

## 7. Caching Strategy — Cache-Aside Pattern

```
Request validation data (dealer, service, vehicle)
        │
        ▼
Redis cache hit? ──YES──→ Return immediately (~1ms)
        │
        NO
        ▼
Query database (~5–50ms)
        │
        ▼
Store result in Redis with TTL
        │
        ▼
Return to caller
```

### Cache Invalidation

Cache must be invalidated **immediately on write**, not on TTL expiry:

```
Dealership deactivated  → del dealer:valid:{id}
Service removed         → del dealer:services:{dealershipId}
Vehicle unregistered    → del vehicle:registered:{vehicleId}
```

---

## 8. Architecture Summary

```
Client Request
      │
      ▼
  API Gateway
      │
      ▼
Validation Layer  ──────────→  Redis Cache
      │ (cache miss)                 ↑
      └──────────────→  PostgreSQL ──┘
                              │
Availability Query  ──────────┘ (always DB, no cache)
      │
      ▼
Booking Service  ────→  appointments table (EXCLUDE gist lock) + invalidate cache
```

---

## 9. Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Slot unit | 30 minutes | Industry standard, flexible for multi-slot services |
| Availability method | Exclusion-based | Simpler, unlimited range, always accurate |
| Pre-compute / sliding window | Not used | Unnecessary complexity for dealership-scale traffic |
| Race condition handling | EXCLUDE gist on appointments | Overlap-safe, no extra table needed for direct booking |
| Reservation table | Not used (future option) | Only needed if a payment/confirmation step is introduced |
| Cache pattern | Cache-Aside | Safe, simple, invalidate on write |
| Slot data | Always live DB | Real-time accuracy required |
| Stable lookup data | Redis cache | Rarely changes, high read frequency |

---

## 10. Observability — OpenTelemetry

### Overview

The system uses the [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/) to provide distributed tracing, metrics, and structured logging across the HTTP layer and usecase layer. All three pillars are initialised at startup before any other component and shut down gracefully on `SIGINT`/`SIGTERM` so no telemetry data is lost.

---

### Initialisation (`infrastructure/telemetry/otel.go`)

`SetupOTel(ctx)` is called at the very beginning of `main()`, before the database, cache, or HTTP server are created. It returns a `shutdown` function that is deferred and called on process exit.

```
main()
  │
  ├─ SetupOTel(ctx)
  │    ├─ Propagator  → W3C TraceContext + Baggage  (registered as global)
  │    ├─ TracerProvider → stdout exporter (batched)  (registered as global)
  │    ├─ MeterProvider  → stdout exporter (periodic) (registered as global)
  │    └─ LoggerProvider → stdout exporter (batched)  (registered as global)
  │
  ├─ DB / Redis / Repos / UseCases ...
  └─ HTTP server starts
```

All three providers use **stdout exporters** suitable for development and log-aggregation pipelines (e.g. forwarded to a collector). Swapping to an OTLP exporter requires only changing the exporter in `SetupOTel` — no other code changes needed.

---

### HTTP Layer (`internal/adapter/http/middleware.go`)

`OTelMiddleware("keyloop-api")` is registered as the first Gin middleware in `RegisterRoutes`. It wraps every inbound HTTP request:

```
Inbound request
      │
      ▼
Extract W3C traceparent / baggage headers
      │
      ▼
Start server-kind span  (tentative name = URL path)
      │
      ▼
c.Next()  ── handler runs ──▶  downstream usecase span (child)
      │
      ▼
Rename span to "METHOD /route/pattern"  (c.FullPath() post-routing)
Set attributes:
  http.request.method   = GET / POST
  http.route            = /api/v1/appointments/available-slots
  http.response.status_code = 200 / 422 / 500
Set status codes.Error on 5xx
span.End()
```

The span name uses the matched **route pattern** (not the raw URL) so high-cardinality paths like `/api/v1/appointments/available-slots?dealership_id=…` are collapsed into a single trace group.

---

### Usecase Layer (`internal/usecase/book-appointment.go`)

A `trace.Tracer` is stored on `AppoinmentBookingUseCase` and initialised once in the constructor via `otel.Tracer("keyloop-test/usecase")`. Each public method creates a **child span** that is automatically linked to the parent HTTP span through the propagated context.

| Method | Span name | Key attributes |
|---|---|---|
| `BookAppointment` | `BookAppointment` | `dealership.id`, `vehicle.id`, `services.count` |
| `AvailableSlots` | `AvailableSlots` | `dealership.id`, `vehicle.id`, `services.count`, `desired_date` |
| `ListAppointments` | `ListAppointments` | `customer.id`, `dealership.id` |

All three methods use the **named-return + deferred error recording** pattern so no error-return path needs manual instrumentation:

```go
func (uc *...) BookAppointment(ctx, req) (_ *Output, err error) {
    ctx, span := uc.tracer.Start(ctx, "BookAppointment", ...)
    defer func() {
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }
        span.End()
    }()
    // ... business logic unchanged
}
```

---

### Trace Propagation

```
Client (with traceparent header)
        │
        ▼
OTelMiddleware  — extracts context, starts root span
        │  (ctx passed through gin → handler → usecase)
        ▼
BookAppointment span  (child of HTTP span)
        │
        └─ future: DB/Redis spans can be added as further children
```

If the client does not send a `traceparent` header, a new root trace is created automatically.

---

### Extending Observability

| What to add | Where |
|---|---|
| Database query spans | Wrap `sql.DB` with `otelsql` driver or add manual spans in each repository method |
| Redis spans | Add manual spans in `infrastructure/redis` cache methods |
| OTLP export (Jaeger, Tempo, etc.) | Replace `stdouttrace.New` in `SetupOTel` with `otlptracegrpc.New` |
| Custom business metrics | Use `otel.Meter("keyloop-test/usecase")` to record counters/histograms (e.g. bookings per dealership) |
| Structured log correlation | Inject `trace_id` / `span_id` into log fields using `trace.SpanFromContext(ctx)` |

---

## 11. When to Revisit These Decisions

- **Add reservation table** if a payment or explicit user-confirmation step is introduced between slot selection and final booking
- **Add pre-compute + sliding window** if the system scales to thousands of simultaneous availability checks across hundreds of dealerships
- **Add read replicas** for the availability query if DB read load becomes a bottleneck
- **Add a queue (e.g. Redis pub/sub)** for cache invalidation if multiple services need to react to booking events
