//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"keyloop-test/internal/domain"
)

// testDB is shared across all tests in this package.
var testDB *sql.DB

// fixtures holds IDs created once in TestMain and reused by all tests.
type fixtures struct {
	// dealership1 has exactly 1 bay and 1 technician — capacity = 1 per slot
	dealership1 string
	// dealership2 has 2 bays and 2 technicians — capacity = 2 per slot
	dealership2 string
	// serviceID is the UUID of the oil-change service row in the services table
	serviceID string
	// customers[i] / vehicles[i] pairs, one per concurrent worker
	customers []string
	vehicles  []string
}

var fix fixtures

const nWorkers = 10

// ─── TestMain ────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15.3",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: cannot start Postgres container (is Docker running?): %v\n", err)
		os.Exit(0)
	}
	defer container.Terminate(ctx) //nolint:errcheck

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err := runMigrations(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "migrations: %v\n", err)
		os.Exit(1)
	}
	if err := seedFixtures(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "seed fixtures: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// runMigrations reads and executes the init-schema migration file.
func runMigrations(db *sql.DB) error {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "../..")
	sql, err := os.ReadFile(filepath.Join(root, "migration/000001_init_schema.up.sql"))
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = db.Exec(string(sql))
	return err
}

// seedFixtures inserts the shared rows used by all tests.
// Each test cleans up appointments in t.Cleanup — these rows are never deleted.
func seedFixtures(db *sql.DB) error {
	fix.serviceID = uuid.New().String()
	if _, err := db.Exec(
		`INSERT INTO services (id, name, description, estimated_minutes, price, is_active)
		 VALUES ($1, 'Oil Change', 'Synthetic oil change', 30, 29.99, true)`,
		fix.serviceID,
	); err != nil {
		return fmt.Errorf("service: %w", err)
	}

	// dealership1: 1 bay, 1 tech → capacity = 1
	fix.dealership1 = uuid.New().String()
	if err := insertDealership(db, fix.dealership1); err != nil {
		return err
	}
	if err := insertBay(db, uuid.New().String(), fix.dealership1, 1); err != nil {
		return err
	}
	if err := insertTechnician(db, uuid.New().String(), fix.dealership1); err != nil {
		return err
	}

	// dealership2: 2 bays, 2 techs → capacity = 2
	fix.dealership2 = uuid.New().String()
	if err := insertDealership(db, fix.dealership2); err != nil {
		return err
	}
	for i := range 2 {
		if err := insertBay(db, uuid.New().String(), fix.dealership2, i+1); err != nil {
			return err
		}
		if err := insertTechnician(db, uuid.New().String(), fix.dealership2); err != nil {
			return err
		}
	}

	// nWorkers customers + vehicles (one per concurrent goroutine)
	fix.customers = make([]string, nWorkers)
	fix.vehicles = make([]string, nWorkers)
	for i := range nWorkers {
		cid := uuid.New().String()
		fix.customers[i] = cid
		if _, err := db.Exec(
			`INSERT INTO customers (id, first_name, last_name, email, phone)
			 VALUES ($1, 'Test', 'User', $2, '0000000000')`,
			cid, fmt.Sprintf("test%d@example.com", i),
		); err != nil {
			return fmt.Errorf("customer %d: %w", i, err)
		}
		vid := uuid.New().String()
		fix.vehicles[i] = vid
		if _, err := db.Exec(
			`INSERT INTO vehicles (id, customer_id, make, model, year, vin, license_plate)
			 VALUES ($1, $2, 'Test', 'Car', 2024, $3, $4)`,
			vid, cid, fmt.Sprintf("ITESTVIN%04d", i), fmt.Sprintf("TEST %04d", i),
		); err != nil {
			return fmt.Errorf("vehicle %d: %w", i, err)
		}
	}
	return nil
}

func insertDealership(db *sql.DB, id string) error {
	_, err := db.Exec(
		`INSERT INTO dealerships (id, name, address, city, phone, is_active, open_time, close_time, working_days)
		 VALUES ($1, 'Test Dealership', '1 Test St', 'TestCity', '0000000000', true, '08:00:00', '18:00:00', ARRAY[0,1,2,3,4,5,6])`,
		id,
	)
	return err
}

func insertBay(db *sql.DB, id, dealershipID string, bayNumber int) error {
	_, err := db.Exec(
		`INSERT INTO service_bays (id, dealership_id, name, bay_number, status)
		 VALUES ($1, $2, $3, $4, 'active')`,
		id, dealershipID, fmt.Sprintf("Bay %d", bayNumber), bayNumber,
	)
	return err
}

// insertTechnician inserts a technician skilled for fix.serviceID.
func insertTechnician(db *sql.DB, id, dealershipID string) error {
	if _, err := db.Exec(
		`INSERT INTO technicians (id, dealership_id, first_name, last_name, status)
		 VALUES ($1, $2, 'Tech', 'Test', 'active')`,
		id, dealershipID,
	); err != nil {
		return fmt.Errorf("technician: %w", err)
	}
	// technician_skills.service_id references services(id) — use raw column name
	if _, err := db.Exec(
		`INSERT INTO technician_skills (technician_id, service_id) VALUES ($1, $2)`,
		id, fix.serviceID,
	); err != nil {
		return fmt.Errorf("technician skill: %w", err)
	}
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// cleanAppointments deletes all rows from the appointments table.
func cleanAppointments(t *testing.T) {
	t.Helper()
	if _, err := testDB.Exec(`DELETE FROM appointments`); err != nil {
		t.Errorf("cleanup appointments: %v", err)
	}
}

// slotAt returns a UTC time at hour:00 tomorrow.
func slotAt(hour int) time.Time {
	now := time.Now().UTC().Add(24 * time.Hour)
	return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
}

// oilChangeServices returns a ServiceSnapshot slice for the shared oil-change service.
func oilChangeServices() []*domain.ServiceSnapshot {
	return []*domain.ServiceSnapshot{{
		ServiceID:        fix.serviceID,
		Name:             "Oil Change",
		EstimatedMinutes: 30,
		Price:            29.99,
	}}
}

// bookingRejected returns true if err represents "no slot available" or an
// exclusion-constraint violation — both mean the booking was correctly refused.
// The repository maps both raw DB signals to domain sentinels before returning:
//   - sql.ErrNoRows (CTE found no free pair)  → domain.ErrNoAvailableResources
//   - pq.Error{23P01} (GIST exclusion fired)  → domain.ErrTimeSlotConflict
func bookingRejected(err error) bool {
	return errors.Is(err, domain.ErrNoAvailableResources) ||
		errors.Is(err, domain.ErrTimeSlotConflict)
}

// // ─── Tests ───────────────────────────────────────────────────────────────────

// // TestCreateAppointment_Sequential_SecondFails checks that a second booking for
// // the exact same time window is rejected when capacity = 1.
// func TestCreateAppointment_Sequential_SecondFails(t *testing.T) {
// 	t.Cleanup(func() { cleanAppointments(t) })

// 	repo := postgrerepository.NewAppointmentRepository(testDB)
// 	start, end := slotAt(10), slotAt(10).Add(30*time.Minute)

// 	_, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
// 		CustomerID: fix.customers[0], VehicleID: fix.vehicles[0],
// 		DealershipID: fix.dealership1,
// 		StartTime:    start, EndTime: end,
// 		Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 	})
// 	if err != nil {
// 		t.Fatalf("first booking failed: %v", err)
// 	}

// 	_, err = repo.CreateAppointment(context.Background(), &domain.Appointment{
// 		CustomerID: fix.customers[1], VehicleID: fix.vehicles[1],
// 		DealershipID: fix.dealership1,
// 		StartTime:    start, EndTime: end,
// 		Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 	})
// 	if !bookingRejected(err) {
// 		t.Fatalf("second booking should have been rejected; got: %v", err)
// 	}
// }

// // TestCreateAppointment_Sequential_DifferentSlots_BothSucceed verifies that
// // sequential, non-overlapping bookings both succeed on the same bay/tech.
// func TestCreateAppointment_Sequential_DifferentSlots_BothSucceed(t *testing.T) {
// 	t.Cleanup(func() { cleanAppointments(t) })

// 	repo := postgrerepository.NewAppointmentRepository(testDB)

// 	for i, hour := range []int{10, 11} { // 10:00-10:30 and 11:00-11:30 — no overlap
// 		start := slotAt(hour)
// 		_, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
// 			CustomerID: fix.customers[i], VehicleID: fix.vehicles[i],
// 			DealershipID: fix.dealership1,
// 			StartTime:    start, EndTime: start.Add(30 * time.Minute),
// 			Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 		})
// 		if err != nil {
// 			t.Errorf("slot %d:00 booking failed: %v", hour, err)
// 		}
// 	}
// }

// // TestCreateAppointment_Sequential_AdjacentSlots_BothSucceed verifies that
// // a slot starting exactly at the previous appointment's end time is allowed
// // (the GIST range is exclusive at the upper bound: [start, end) ).
// func TestCreateAppointment_Sequential_AdjacentSlots_BothSucceed(t *testing.T) {
// 	t.Cleanup(func() { cleanAppointments(t) })

// 	repo := postgrerepository.NewAppointmentRepository(testDB)
// 	first := slotAt(10)
// 	second := first.Add(30 * time.Minute) // starts exactly when first ends

// 	for i, start := range []time.Time{first, second} {
// 		_, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
// 			CustomerID: fix.customers[i], VehicleID: fix.vehicles[i],
// 			DealershipID: fix.dealership1,
// 			StartTime:    start, EndTime: start.Add(30 * time.Minute),
// 			Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 		})
// 		if err != nil {
// 			t.Errorf("adjacent slot %d booking failed: %v", i, err)
// 		}
// 	}
// }

// // TestCreateAppointment_Concurrent_CapacityOne_ExactlyOneSucceeds fires two
// // goroutines at the same slot simultaneously. With 1 bay and 1 technician,
// // exactly one booking must succeed — the other is rejected either by the CTE
// // (returns 0 rows) or by the GIST exclusion constraint (error code 23P01).
// func TestCreateAppointment_Concurrent_CapacityOne_ExactlyOneSucceeds(t *testing.T) {
// 	t.Cleanup(func() { cleanAppointments(t) })

// 	repo := postgrerepository.NewAppointmentRepository(testDB)
// 	start, end := slotAt(10), slotAt(10).Add(30*time.Minute)

// 	type result struct {
// 		appt *domain.Appointment
// 		err  error
// 	}
// 	results := make([]result, 2)

// 	var ready sync.WaitGroup
// 	var gate sync.WaitGroup
// 	var done sync.WaitGroup
// 	ready.Add(2)
// 	gate.Add(1)
// 	done.Add(2)

// 	for i := range 2 {
// 		go func(i int) {
// 			defer done.Done()
// 			ready.Done() // signal: parked at gate
// 			gate.Wait()  // released simultaneously with the other goroutine
// 			appt, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
// 				CustomerID: fix.customers[i], VehicleID: fix.vehicles[i],
// 				DealershipID: fix.dealership1,
// 				StartTime:    start, EndTime: end,
// 				Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 			})
// 			results[i] = result{appt, err}
// 		}(i)
// 	}

// 	ready.Wait() // both goroutines are at the gate
// 	gate.Done()  // release simultaneously
// 	done.Wait()

// 	successes := 0
// 	for i, r := range results {
// 		if r.err == nil {
// 			successes++
// 		} else if !bookingRejected(r.err) {
// 			t.Errorf("worker %d: unexpected error (not a booking rejection): %v", i, r.err)
// 		}
// 	}
// 	if successes != 1 {
// 		t.Errorf("capacity=1: expected exactly 1 success, got %d", successes)
// 	}
// }

// // TestCreateAppointment_Concurrent_CapacityTwo_BothSucceed fires two goroutines
// // simultaneously at the same slot on dealership2 (2 bays, 2 techs).
// // Both must succeed and receive different bays and technicians.
// func TestCreateAppointment_Concurrent_CapacityTwo_BothSucceed(t *testing.T) {
// 	t.Cleanup(func() { cleanAppointments(t) })

// 	repo := postgrerepository.NewAppointmentRepository(testDB)
// 	start, end := slotAt(10), slotAt(10).Add(30*time.Minute)

// 	type result struct {
// 		appt *domain.Appointment
// 		err  error
// 	}
// 	results := make([]result, 2)

// 	var ready, gate, done sync.WaitGroup
// 	ready.Add(2)
// 	gate.Add(1)
// 	done.Add(2)

// 	for i := range 2 {
// 		go func(i int) {
// 			defer done.Done()
// 			ready.Done()
// 			gate.Wait()
// 			appt, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
// 				CustomerID: fix.customers[i], VehicleID: fix.vehicles[i],
// 				DealershipID: fix.dealership2, // capacity = 2
// 				StartTime:    start, EndTime: end,
// 				Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 			})
// 			results[i] = result{appt, err}
// 		}(i)
// 	}

// 	ready.Wait()
// 	gate.Done()
// 	done.Wait()

// 	for i, r := range results {
// 		if r.err != nil {
// 			t.Errorf("worker %d: expected success with capacity=2, got: %v", i, r.err)
// 		}
// 	}
// 	if results[0].appt != nil && results[1].appt != nil {
// 		if results[0].appt.ServiceBayID == results[1].appt.ServiceBayID {
// 			t.Error("both bookings were assigned the same bay — concurrent writes corrupted slot allocation")
// 		}
// 		if results[0].appt.TechnicianID == results[1].appt.TechnicianID {
// 			t.Error("both bookings were assigned the same technician — concurrent writes corrupted slot allocation")
// 		}
// 	}
// }

// // TestCreateAppointment_Concurrent_TenWorkers_ExactlyOneSucceeds fires 10
// // goroutines simultaneously at the same slot with capacity=1. Exactly one must
// // win regardless of scheduling order — this stresses both the CTE "read before
// // write" path and the GIST exclusion constraint fallback.
// func TestCreateAppointment_Concurrent_TenWorkers_ExactlyOneSucceeds(t *testing.T) {
// 	t.Cleanup(func() { cleanAppointments(t) })

// 	repo := postgrerepository.NewAppointmentRepository(testDB)
// 	start, end := slotAt(10), slotAt(10).Add(30*time.Minute)

// 	type result struct {
// 		appt *domain.Appointment
// 		err  error
// 	}
// 	results := make([]result, nWorkers)

// 	var ready, gate, done sync.WaitGroup
// 	ready.Add(nWorkers)
// 	gate.Add(1)
// 	done.Add(nWorkers)

// 	for i := range nWorkers {
// 		go func(i int) {
// 			defer done.Done()
// 			ready.Done()
// 			gate.Wait()
// 			appt, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
// 				CustomerID: fix.customers[i], VehicleID: fix.vehicles[i],
// 				DealershipID: fix.dealership1,
// 				StartTime:    start, EndTime: end,
// 				Services: oilChangeServices(), Status: domain.AppointmentStatusPending,
// 			})
// 			results[i] = result{appt, err}
// 		}(i)
// 	}

// 	ready.Wait()
// 	gate.Done()
// 	done.Wait()

// 	successes := 0
// 	for i, r := range results {
// 		if r.err == nil {
// 			successes++
// 		} else if !bookingRejected(r.err) {
// 			t.Errorf("worker %d: unexpected error: %v", i, r.err)
// 		}
// 	}
// 	if successes != 1 {
// 		t.Errorf("capacity=1, %d workers: expected exactly 1 success, got %d", nWorkers, successes)
// 	}
// }
