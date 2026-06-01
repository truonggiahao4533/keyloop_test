DELETE FROM vehicles   WHERE id LIKE 'ffffffff-0001%';
DELETE FROM customers  WHERE id LIKE 'eeeeeeee-0001%';
DELETE FROM technician_skills
  WHERE technician_id IN (
    SELECT id FROM technicians WHERE id LIKE 'dddddddd-0001%'
  );
DELETE FROM technicians        WHERE id LIKE 'dddddddd-0001%';
DELETE FROM services WHERE id LIKE 'cccccccc-0001%';
DELETE FROM service_bays       WHERE id LIKE 'bbbbbbbb-0001%';
DELETE FROM dealerships        WHERE id LIKE 'aaaaaaaa-0001%';
