CREATE TYPE bay_status AS ENUM (
    'active',
    'inactive',
    'maintenance'
);

CREATE TYPE technician_status AS ENUM (
    'active',
    'inactive',
    'on_leave'
);

CREATE TYPE appointment_status AS ENUM (
    'pending',
    'confirmed',
    'cancelled',
    'completed',
    'no_show'
);

-- CREATE TYPE service_type AS ENUM (
--     'oil_change',
--     'tire_rotation',
--     'brake_inspection',
--     'engine_diagnostics',
--     'full_service',
--     'mot_inspection'
-- );

CREATE TABLE customers (
    id UUID PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20) NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE vehicles (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    make VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    year INT NOT NULL,
    vin VARCHAR(50) NOT NULL UNIQUE,
    license_plate VARCHAR(20) NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE dealerships (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    open_time TIME NOT NULL DEFAULT '09:00:00',
    close_time TIME NOT NULL DEFAULT '17:00:00',
    deleted_at TIMESTAMP WITH TIME ZONE NULL ,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE service_bays (
    id UUID PRIMARY KEY,
    dealership_id UUID NOT NULL REFERENCES dealerships(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    bay_number INT NOT NULL,
    status bay_status NOT NULL DEFAULT 'active',
    deleted_at TIMESTAMP WITH TIME ZONE NULL ,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(dealership_id, bay_number)
);

CREATE TABLE service_definitions (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    estimated_minutes INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE technicians (
    id UUID PRIMARY KEY,
    dealership_id UUID NOT NULL REFERENCES dealerships(id) ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    status technician_status NOT NULL DEFAULT 'active',
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE technician_skills (
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    service_definition_id UUID NOT NULL REFERENCES service_definitions(id) ON DELETE CASCADE,
    PRIMARY KEY (technician_id, service_definition_id)
);


CREATE TABLE appointments (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    dealership_id UUID NOT NULL REFERENCES dealerships(id) ON DELETE CASCADE,
    service_bay_id UUID NOT NULL REFERENCES service_bays(id) ON DELETE CASCADE,
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    services JSONB NOT NULL,
    status appointment_status NOT NULL DEFAULT 'pending',
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    notes TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Prevent overlapping bookings for the same bay (pending and confirmed only)
    CONSTRAINT no_overlap_bay EXCLUDE USING gist (
        service_bay_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (deleted_at IS NULL AND status IN ('pending', 'confirmed')),

    -- Prevent overlapping bookings for the same technician
    CONSTRAINT no_overlap_tech EXCLUDE USING gist (
        technician_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (deleted_at IS NULL AND status IN ('pending', 'confirmed'))
);

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE INDEX idx_appointments_start_time ON appointments(start_time);
CREATE INDEX idx_appointments_end_time ON appointments(end_time);
CREATE INDEX idx_appointments_dealership_id ON appointments(dealership_id);





