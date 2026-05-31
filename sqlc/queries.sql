-- name: GetAvailableSlots :many
SELECT slot::TIMESTAMP WITH TIME ZONE
FROM generate_series(
  sqlc.arg(start_datetime)::TIMESTAMP WITH TIME ZONE,
  sqlc.arg(end_datetime)::TIMESTAMP WITH TIME ZONE,
  '30 minutes'::INTERVAL
) AS slot
WHERE
  -- At least 1 bay is free for the full duration
  EXISTS (
    SELECT 1 FROM service_bays b
    WHERE b.dealership_id = sqlc.arg(dealership_id)
    AND b.id NOT IN (
      SELECT service_bay_id FROM appointments
      WHERE status IN ('pending', 'confirmed')
      AND tstzrange(start_time, end_time) &&
            tstzrange(slot, slot + sqlc.arg(duration)::INTERVAL)
    )
  )
  AND
  -- At least 1 technician who has ALL required skills and is free for the full duration
  EXISTS (
    SELECT 1 FROM technicians t
    WHERE t.dealership_id = sqlc.arg(dealership_id)
    AND t.id IN (
      -- tech must have ALL required skills
      SELECT technician_id FROM technician_skills
      WHERE skill = ANY(sqlc.arg(service_types)::text[])
      GROUP BY technician_id
      HAVING COUNT(DISTINCT skill) = cardinality(sqlc.arg(service_types)::text[])
    )
    AND t.id NOT IN (
      SELECT technician_id FROM appointments
      WHERE status IN ('pending', 'confirmed')
      AND tstzrange(start_time, end_time) &&
            tstzrange(slot, slot + sqlc.arg(duration)::INTERVAL)
    )
  )
ORDER BY slot ASC;

--

-- =========================================================================
-- CUSTOMERS
-- =========================================================================

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = $1;

-- name: ListCustomers :many
SELECT * FROM customers ORDER BY created_at DESC;

-- name: CreateCustomer :one
INSERT INTO customers (
  id, first_name, last_name, email, phone
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET first_name = $2,
    last_name = $3,
    email = $4,
    phone = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;


-- =========================================================================
-- VEHICLES
-- =========================================================================

-- name: GetVehicle :one
SELECT * FROM vehicles WHERE id = $1;

-- name: ListVehiclesByCustomer :many
SELECT * FROM vehicles WHERE customer_id = $1 ORDER BY created_at DESC;

-- name: CreateVehicle :one
INSERT INTO vehicles (
  id, customer_id, make, model, year, vin, license_plate
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateVehicle :one
UPDATE vehicles
SET make = $2,
    model = $3,
    year = $4,
    vin = $5,
    license_plate = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteVehicle :exec
DELETE FROM vehicles WHERE id = $1;


-- =========================================================================
-- DEALERSHIPS
-- =========================================================================

-- name: GetDealership :one
SELECT * FROM dealerships WHERE id = $1;

-- name: ListDealerships :many
SELECT * FROM dealerships ORDER BY name ASC;

-- name: CreateDealership :one
INSERT INTO dealerships (
  id, name, address, city, phone, is_active, open_time, close_time
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateDealership :one
UPDATE dealerships
SET name = $2,
    address = $3,
    city = $4,
    phone = $5,
    is_active = $6,
    open_time = $7,
    close_time = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDealership :exec
DELETE FROM dealerships WHERE id = $1;


-- =========================================================================
-- SERVICE BAYS
-- =========================================================================

-- name: GetServiceBay :one
SELECT * FROM service_bays WHERE id = $1;

-- name: ListServiceBaysByDealership :many
SELECT * FROM service_bays WHERE dealership_id = $1 ORDER BY bay_number ASC;

-- name: CreateServiceBay :one
INSERT INTO service_bays (
  id, dealership_id, name, bay_number, status
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateServiceBay :one
UPDATE service_bays
SET name = $2,
    bay_number = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteServiceBay :exec
DELETE FROM service_bays WHERE id = $1;


-- =========================================================================
-- SERVICE DEFINITIONS
-- =========================================================================

-- name: GetServiceDefinition :one
SELECT * FROM service_definitions WHERE id = $1;

-- name: ListServiceDefinitions :many
SELECT * FROM service_definitions ORDER BY name ASC;

-- name: CreateServiceDefinition :one
INSERT INTO service_definitions (
  id, name, description, estimated_minutes, price, is_active
) VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateServiceDefinition :one
UPDATE service_definitions
SET name = $2,
    description = $3,
    estimated_minutes = $4,
    price = $5,
    is_active = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteServiceDefinition :exec
DELETE FROM service_definitions WHERE id = $1;


-- =========================================================================
-- TECHNICIANS
-- =========================================================================

-- name: GetTechnician :one
SELECT * FROM technicians WHERE id = $1;

-- name: ListTechniciansByDealership :many
SELECT * FROM technicians WHERE dealership_id = $1 ORDER BY first_name ASC;

-- name: CreateTechnician :one
INSERT INTO technicians (
  id, dealership_id, first_name, last_name, status
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateTechnician :one
UPDATE technicians
SET first_name = $2,
    last_name = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTechnician :exec
DELETE FROM technicians WHERE id = $1;


-- =========================================================================
-- TECHNICIAN SKILLS
-- =========================================================================

-- name: ListTechnicianSkills :many
SELECT * FROM technician_skills WHERE technician_id = $1;

-- name: CreateTechnicianSkill :exec
INSERT INTO technician_skills (
  technician_id, service_definition_id
) VALUES (
  $1, $2
);

-- name: DeleteTechnicianSkill :exec
DELETE FROM technician_skills WHERE technician_id = $1 AND service_definition_id = $2;


-- =========================================================================
-- APPOINTMENTS (Remaining CRUD)
-- =========================================================================

-- name: GetAppointment :one
SELECT * FROM appointments WHERE id = $1;

-- name: ListAppointmentsByDealership :many
SELECT * FROM appointments WHERE dealership_id = $1 ORDER BY start_time DESC;

-- name: ListAppointmentsByCustomer :many
SELECT * FROM appointments WHERE customer_id = $1 ORDER BY start_time DESC;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- Service_ids input to fetch technicians with all required skills and to check bay/tech availability
-- Services input to store snapshot of service details at time of booking (name, price, duration) to prevent issues if service definitions change later 

-- name: CreateAppointment :one
WITH slot_range AS (
  SELECT tstzrange(
    sqlc.arg(start_time)::TIMESTAMPTZ,
    sqlc.arg(start_time)::TIMESTAMPTZ + sqlc.arg(duration)::INTERVAL
  ) AS range
),
qualified_technicians AS (
  SELECT t.id
  FROM technicians t
  INNER JOIN technician_skills ts ON ts.technician_id = t.id
  WHERE t.dealership_id = sqlc.arg(dealership_id)
    AND ts.skill = ANY(sqlc.arg(service_ids)::text  [])
  GROUP BY t.id
  HAVING COUNT(DISTINCT ts.skill) = cardinality(sqlc.arg(service_ids)::text[])
),
available_bay AS (
  SELECT b.id
  FROM service_bays b, slot_range sr
  WHERE b.dealership_id = sqlc.arg(dealership_id)
    AND NOT EXISTS (
      SELECT 1 FROM appointments a
      WHERE a.bay_id = b.id
        AND a.dealership_id = sqlc.arg(dealership_id)
        AND a.status <> 'CANCELLED'
        AND tstzrange(a.start_time, a.end_time) && sr.range
    )
  LIMIT 1
),
available_technician AS (
  SELECT qt.id
  FROM qualified_technicians qt, slot_range sr
  WHERE NOT EXISTS (
      SELECT 1 FROM appointments a
      WHERE a.technician_id = qt.id
        AND a.dealership_id = sqlc.arg(dealership_id)
        AND a.status <> 'CANCELLED'
        AND tstzrange(a.start_time, a.end_time) && sr.range
    )
  LIMIT 1
)
INSERT INTO appointments (
  dealership_id, customer_id, vehicle_id,
  bay_id, technician_id,
  start_time, end_time,
  status, notes, services
)
SELECT
  sqlc.arg(dealership_id),
  sqlc.arg(customer_id),
  sqlc.arg(vehicle_id),
  availbay.id,
  availtech.id,
  sqlc.arg(start_time)::TIMESTAMPTZ,
  sqlc.arg(start_time)::TIMESTAMPTZ + sqlc.arg(duration)::INTERVAL,
  sqlc.arg(status)::appointment_status,
  sqlc.arg(notes),
  sqlc.arg(services)::JSONB

FROM available_bay availbay, available_technician availtech
RETURNING *;

-- name: DeleteAppointment :exec
DELETE FROM appointments WHERE id = $1;


