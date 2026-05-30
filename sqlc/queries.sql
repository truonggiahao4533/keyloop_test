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
      SELECT bay_id FROM appointments
      WHERE tstzrange(start_time, end_time) &&
            tstzrange(slot, slot + sqlc.arg(duration)::INTERVAL)
    )
    AND b.id NOT IN (
      SELECT bay_id FROM reservations
      WHERE status = 'PENDING' AND expires_at > NOW()
      AND tstzrange(start_time, end_time) &&
            tstzrange(slot, slot + sqlc.arg(duration)::INTERVAL)
    )
  )
  AND
  -- At least 1 qualified technician is free for the full duration
  EXISTS (
    SELECT 1 FROM technicians t
    JOIN technician_skills q ON q.technician_id = t.id
    WHERE t.dealership_id = sqlc.arg(dealership_id)
    AND q.skill = sqlc.arg(service_type)
    AND t.id NOT IN (
      SELECT technician_id FROM appointments
      WHERE tstzrange(start_time, end_time) &&
            tstzrange(slot, slot + sqlc.arg(duration)::INTERVAL)
    )
    AND t.id NOT IN (
      SELECT technician_id FROM reservations
      WHERE status = 'PENDING' AND expires_at > NOW()
      AND tstzrange(start_time, end_time) &&
            tstzrange(slot, slot + sqlc.arg(duration)::INTERVAL)
    )
  );

-- name: CreateReservation :one
INSERT INTO reservations (
  id, bay_id, technician_id, start_time, end_time, user_id, expires_at, status
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: DeleteExpiredReservations :exec
DELETE FROM reservations
WHERE expires_at < NOW()
AND status = 'PENDING';

-- name: DeleteReservation :exec
DELETE FROM reservations
WHERE id = $1;

-- name: CreateAppointment :one
INSERT INTO appointments (
  id, customer_id, vehicle_id, dealership_id, service_bay_id, technician_id, service_type, status, start_time, end_time, notes
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;


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
  id, name, address, city, phone, is_active
) VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateDealership :one
UPDATE dealerships
SET name = $2,
    address = $3,
    city = $4,
    phone = $5,
    is_active = $6,
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
  id, name, type, description, estimated_minutes, is_active
) VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateServiceDefinition :one
UPDATE service_definitions
SET name = $2,
    type = $3,
    description = $4,
    estimated_minutes = $5,
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
  technician_id, skill
) VALUES (
  $1, $2
);

-- name: DeleteTechnicianSkill :exec
DELETE FROM technician_skills WHERE technician_id = $1 AND skill = $2;


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

-- name: DeleteAppointment :exec
DELETE FROM appointments WHERE id = $1;


-- =========================================================================
-- RESERVATIONS (Remaining CRUD)
-- =========================================================================

-- name: GetReservation :one
SELECT * FROM reservations WHERE id = $1;

-- name: ListReservationsByDealership :many
SELECT r.* 
FROM reservations r
JOIN service_bays b ON r.bay_id = b.id
WHERE b.dealership_id = $1
ORDER BY r.start_time DESC;

-- name: UpdateReservationStatus :one
UPDATE reservations
SET status = $2
WHERE id = $1
RETURNING *;
