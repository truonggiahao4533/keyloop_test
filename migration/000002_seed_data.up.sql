-- =========================================================================
-- Dealerships
-- =========================================================================
INSERT INTO dealerships (id, name, address, city, phone, is_active, open_time, close_time) VALUES
  ('aaaaaaaa-0001-0001-0001-000000000001', 'Keyloop Motors London',    '12 Baker Street',   'London',     '+44 20 7946 0001', TRUE, '08:00', '18:00'),
  ('aaaaaaaa-0001-0001-0001-000000000002', 'Keyloop Motors Manchester', '88 Deansgate',      'Manchester', '+44 16 1946 0002', TRUE, '08:00', '18:00');

-- =========================================================================
-- Service Bays
-- =========================================================================
INSERT INTO service_bays (id, dealership_id, name, bay_number, status) VALUES
  -- London
  ('bbbbbbbb-0001-0001-0001-000000000001', 'aaaaaaaa-0001-0001-0001-000000000001', 'Bay 1', 1, 'active'),
  ('bbbbbbbb-0001-0001-0001-000000000002', 'aaaaaaaa-0001-0001-0001-000000000001', 'Bay 2', 2, 'active'),
  ('bbbbbbbb-0001-0001-0001-000000000003', 'aaaaaaaa-0001-0001-0001-000000000001', 'Bay 3', 3, 'maintenance'),
  -- Manchester
  ('bbbbbbbb-0001-0001-0001-000000000004', 'aaaaaaaa-0001-0001-0001-000000000002', 'Bay 1', 1, 'active'),
  ('bbbbbbbb-0001-0001-0001-000000000005', 'aaaaaaaa-0001-0001-0001-000000000002', 'Bay 2', 2, 'active');

-- =========================================================================
-- Service Definitions
-- =========================================================================
INSERT INTO services (id, name, description, estimated_minutes, price, is_active) VALUES
  ('cccccccc-0001-0001-0001-000000000001', 'Oil Change',          'Full synthetic oil change with filter replacement',          30,  29.99, TRUE),
  ('cccccccc-0001-0001-0001-000000000002', 'Tire Rotation',       'Rotate all four tyres to even out wear',                    45,  19.99, TRUE),
  ('cccccccc-0001-0001-0001-000000000003', 'Brake Inspection',    'Inspect brake pads, discs, callipers and fluid',            60,  39.99, TRUE),
  ('cccccccc-0001-0001-0001-000000000004', 'Engine Diagnostics',  'Full OBD-II diagnostic scan and fault report',              90,  79.99, TRUE),
  ('cccccccc-0001-0001-0001-000000000005', 'Full Service',        'Comprehensive 60-point vehicle health check and service',  120, 149.99, TRUE),
  ('cccccccc-0001-0001-0001-000000000006', 'MOT Inspection',      'Annual MOT roadworthiness test',                            60,  54.99, TRUE);

-- =========================================================================
-- Technicians
-- =========================================================================
INSERT INTO technicians (id, dealership_id, first_name, last_name, status) VALUES
  -- London
  ('dddddddd-0001-0001-0001-000000000001', 'aaaaaaaa-0001-0001-0001-000000000001', 'Alice',   'Hartley',  'active'),
  ('dddddddd-0001-0001-0001-000000000002', 'aaaaaaaa-0001-0001-0001-000000000001', 'Bob',     'Sinclair', 'active'),
  ('dddddddd-0001-0001-0001-000000000003', 'aaaaaaaa-0001-0001-0001-000000000001', 'Charlie', 'Owens',    'active'),
  -- Manchester
  ('dddddddd-0001-0001-0001-000000000004', 'aaaaaaaa-0001-0001-0001-000000000002', 'Diana',   'Patel',    'active'),
  ('dddddddd-0001-0001-0001-000000000005', 'aaaaaaaa-0001-0001-0001-000000000002', 'Evan',    'Brooks',   'active');

-- =========================================================================
-- Technician Skills  (FK → service_definitions)
-- cccccccc-…-01 oil_change | 02 tire_rotation | 03 brake_inspection
-- cccccccc-…-04 engine_diagnostics | 05 full_service | 06 mot_inspection
-- =========================================================================
INSERT INTO technician_skills (technician_id, service_id) VALUES
  -- Alice: routine maintenance
  ('dddddddd-0001-0001-0001-000000000001', 'cccccccc-0001-0001-0001-000000000001'), -- oil_change
  ('dddddddd-0001-0001-0001-000000000001', 'cccccccc-0001-0001-0001-000000000002'), -- tire_rotation
  ('dddddddd-0001-0001-0001-000000000001', 'cccccccc-0001-0001-0001-000000000003'), -- brake_inspection
  -- Bob: diagnostics & full service
  ('dddddddd-0001-0001-0001-000000000002', 'cccccccc-0001-0001-0001-000000000004'), -- engine_diagnostics
  ('dddddddd-0001-0001-0001-000000000002', 'cccccccc-0001-0001-0001-000000000005'), -- full_service
  ('dddddddd-0001-0001-0001-000000000002', 'cccccccc-0001-0001-0001-000000000001'), -- oil_change
  -- Charlie: MOT + routine
  ('dddddddd-0001-0001-0001-000000000003', 'cccccccc-0001-0001-0001-000000000006'), -- mot_inspection
  ('dddddddd-0001-0001-0001-000000000003', 'cccccccc-0001-0001-0001-000000000001'), -- oil_change
  ('dddddddd-0001-0001-0001-000000000003', 'cccccccc-0001-0001-0001-000000000002'), -- tire_rotation
  -- Diana (Manchester): full-service specialist
  ('dddddddd-0001-0001-0001-000000000004', 'cccccccc-0001-0001-0001-000000000001'), -- oil_change
  ('dddddddd-0001-0001-0001-000000000004', 'cccccccc-0001-0001-0001-000000000002'), -- tire_rotation
  ('dddddddd-0001-0001-0001-000000000004', 'cccccccc-0001-0001-0001-000000000003'), -- brake_inspection
  ('dddddddd-0001-0001-0001-000000000004', 'cccccccc-0001-0001-0001-000000000005'), -- full_service
  -- Evan (Manchester): diagnostics & MOT
  ('dddddddd-0001-0001-0001-000000000005', 'cccccccc-0001-0001-0001-000000000004'), -- engine_diagnostics
  ('dddddddd-0001-0001-0001-000000000005', 'cccccccc-0001-0001-0001-000000000006'); -- mot_inspection

-- =========================================================================
-- Customers
-- =========================================================================
INSERT INTO customers (id, first_name, last_name, email, phone) VALUES
  ('eeeeeeee-0001-0001-0001-000000000001', 'James',  'Morgan',   'james.morgan@example.com',   '+44 79 0001 0001'),
  ('eeeeeeee-0001-0001-0001-000000000002', 'Sarah',  'Collins',  'sarah.collins@example.com',  '+44 79 0001 0002'),
  ('eeeeeeee-0001-0001-0001-000000000003', 'Liam',   'Nguyen',   'liam.nguyen@example.com',    '+44 79 0001 0003'),
  ('eeeeeeee-0001-0001-0001-000000000004', 'Emma',   'Kaur',     'emma.kaur@example.com',      '+44 79 0001 0004'),
  ('eeeeeeee-0001-0001-0001-000000000005', 'Oliver', 'Perez',    'oliver.perez@example.com',   '+44 79 0001 0005');

-- =========================================================================
-- Vehicles
-- =========================================================================
INSERT INTO vehicles (id, customer_id, make, model, year, vin, license_plate) VALUES
  ('ffffffff-0001-0001-0001-000000000001', 'eeeeeeee-0001-0001-0001-000000000001', 'Ford',       'Focus',  2021, 'WF0WXXGCDWMS00001', 'LK21 AAA'),
  ('ffffffff-0001-0001-0001-000000000002', 'eeeeeeee-0001-0001-0001-000000000002', 'Toyota',     'Yaris',  2022, 'JTDKB20U023000002', 'SA22 BBB'),
  ('ffffffff-0001-0001-0001-000000000003', 'eeeeeeee-0001-0001-0001-000000000003', 'BMW',        '3 Series', 2020, 'WBA8E9G53HNU00003', 'LN20 CCC'),
  ('ffffffff-0001-0001-0001-000000000004', 'eeeeeeee-0001-0001-0001-000000000004', 'Volkswagen', 'Golf',   2023, 'WVWZZZ1KZDW000004', 'YE23 DDD'),
  ('ffffffff-0001-0001-0001-000000000005', 'eeeeeeee-0001-0001-0001-000000000005', 'Honda',      'Civic',  2019, '2HGFB2F59KH000005', 'OU19 EEE');
