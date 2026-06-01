# Unified Service Scheduler — System Design Summary

## 1. Overview

A dealership appointment booking system that allows users to book vehicle service appointments by checking real-time availability of service bays and qualified technicians, then confirming a persistent appointment record.

---

## 2. Data Classification

| Data | Change Frequency | Strategy |
|---|---|---|
| Dealership active status | Rarely | Cache (TTL: 1–24 hrs) |
| Services offered | Rarely | Cache (TTL: 1–24 hrs) |
| Technician qualifications | Occasionally | Cache (TTL: 1–24 hrs) |
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
  - Dealership
  - Vehicle
  - Service Type
        │
        ▼
[Step 1] Validate inputs
         - Is dealership valid & active?        ← Redis Cache
         - Does dealership offer this service?  ← Redis Cache
         - Is vehicle registered here?          ← Redis Cache
        │
        ▼
[Step 2] Calculate service duration
         e.g. Brake Service = 60 min = 2 consecutive slots
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
[Step 6] INSERT into reservations table         ← Locks the slot (5–10 min TTL)
         │
         ├── FAIL (conflict) → "Slot just taken, pick another"
         │
         SUCCESS → slot is held for this user
        │
        ▼
[Step 7] User confirms on review screen
         │
         ├── User abandons → reservation expires → slot auto-released
         │
         User confirms
         │
         ▼
[Step 8] BEGIN TRANSACTION
           INSERT into appointments
           DELETE from reservations
         COMMIT
```

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

## 6. Race Condition Protection — Reservation Table

### The Problem (TOCTOU Race Condition)

```
User A checks Mon 10:00 → FREE
User B checks Mon 10:00 → FREE
Both click Book simultaneously
  → Both insert appointment
  → DOUBLE BOOKING 💥
```

### The Solution — Reservation Table as a Temporary Lock

```sql
CREATE TABLE reservations (
  id            UUID PRIMARY KEY,
  bay_id        UUID NOT NULL,
  technician_id UUID NOT NULL,
  start_time    TIMESTAMP NOT NULL,
  end_time      TIMESTAMP NOT NULL,
  user_id       UUID NOT NULL,
  expires_at    TIMESTAMP NOT NULL,
  status        VARCHAR(20) DEFAULT 'PENDING',

  -- Prevent overlapping reservations for the same bay
  CONSTRAINT no_overlap_bay EXCLUDE USING gist (
    bay_id WITH =,
    tsrange(start_time, end_time) WITH &&
  ),

  -- Prevent overlapping reservations for the same technician
  CONSTRAINT no_overlap_tech EXCLUDE USING gist (
    technician_id WITH =,
    tsrange(start_time, end_time) WITH &&
  )
);
```

The `EXCLUDE USING gist` constraint handles **overlapping time ranges** — something a plain `UNIQUE` constraint cannot do.

### Reservation Expiry Cleanup

```sql
-- Run every minute via cron job or DB scheduler
DELETE FROM reservations
WHERE expires_at < NOW()
AND status = 'PENDING';
```

Availability queries must also treat expired reservations as free:

```sql
AND (reservations.expires_at > NOW() OR reservations.id IS NULL)
```

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
Reservation Service  ──→  reservations table (EXCLUDE gist lock)
      │
      ▼
Booking Service  ────→  appointments table + invalidate cache
```

---

## 9. Key Design Decisions

| Decision | Choice | Reason |
|---|---|---|
| Slot unit | 30 minutes | Industry standard, flexible for multi-slot services |
| Availability method | Exclusion-based | Simpler, unlimited range, always accurate |
| Pre-compute / sliding window | Not used | Unnecessary complexity for dealership-scale traffic |
| Race condition handling | Reservation table + EXCLUDE gist | Overlap-safe locking with auto-expiry |
| Cache pattern | Cache-Aside | Safe, simple, invalidate on write |
| Slot data | Always live DB | Real-time accuracy required |
| Stable lookup data | Redis cache | Rarely changes, high read frequency |

---

## 10. When to Revisit These Decisions

- **Add pre-compute + sliding window** if the system scales to thousands of simultaneous availability checks across hundreds of dealerships
- **Add read replicas** for the availability query if DB read load becomes a bottleneck
- **Add a queue (e.g. Redis pub/sub)** for cache invalidation if multiple services need to react to booking events