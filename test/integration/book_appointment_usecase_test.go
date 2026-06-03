//go:build integration

package integration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgrerepository "keyloop-test/infrastructure/postgresql/postgre-repository"
	"keyloop-test/internal/domain"
	"keyloop-test/internal/lock"
	"keyloop-test/internal/port"
	"keyloop-test/internal/usecase"
)

// ── no-op tracer / span ───────────────────────────────────────────────────────

type ucNoopTracer struct{}

func (t *ucNoopTracer) Start(ctx context.Context, _ string, _ ...port.SpanAttr) (context.Context, port.Span) {
	return ctx, &ucNoopSpan{}
}

type ucNoopSpan struct{}

func (s *ucNoopSpan) End()                    {}
func (s *ucNoopSpan) RecordError(_ error)     {}
func (s *ucNoopSpan) SetErrorStatus(_ string) {}

// ── no-op service cache (always misses → falls back to the real serviceRepo) ──

type ucNoopServiceCache struct{}

func (c *ucNoopServiceCache) GetService(_ context.Context, _ string) (*domain.Service, bool, error) {
	return nil, false, nil
}
func (c *ucNoopServiceCache) SetService(_ context.Context, _ string, _ *domain.Service, _ time.Duration) error {
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newBookingUC wires the full BookAppointment use case against the shared
// testDB Postgres container and a fresh miniredis instance.
// The miniredis server is stopped in t.Cleanup.
func newBookingUC(t *testing.T) *usecase.AppoinmentBookingUseCase {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err, "start miniredis")
	t.Cleanup(mr.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	return usecase.NewAppointmentBookingUseCase(
		postgrerepository.NewDealershipRepository(testDB),
		postgrerepository.NewServiceBayRepository(testDB),
		postgrerepository.NewTechnicianRepository(testDB),
		postgrerepository.NewServiceRepository(testDB),
		postgrerepository.NewVehicleRepository(testDB),
		&ucNoopServiceCache{},
		postgrerepository.NewAvailabilitySlotRepository(testDB),
		postgrerepository.NewAppointmentRepository(testDB),
		postgrerepository.NewUnifiedRepository(testDB),
		lock.NewBookingLocker(redisClient),
		&ucNoopTracer{},
	)
}

// bookingInput returns a valid input for dealership1 at the given start time.
func bookingInput(customerIdx int, start time.Time) *usecase.AppoinmentBookingInput {
	return &usecase.AppoinmentBookingInput{
		CustomerID:       fix.customers[customerIdx],
		DealershipID:     fix.dealership1,
		VehicleID:        fix.vehicles[customerIdx],
		Services:         []string{fix.serviceID},
		DesiredStartTime: start,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestBookAppointmentUseCase_HappyPath verifies the full 3-step flow succeeds:
// FindAvailableTechnicianAndBay → AcquireLock → InsertAppointment.
func TestBookAppointmentUseCase_HappyPath(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	uc := newBookingUC(t)
	out, err := uc.BookAppointment(context.Background(), bookingInput(0, slotAt(9)))

	require.NoError(t, err)
	assert.NotEmpty(t, out.AppointmentID)
	assert.Equal(t, 30, out.DurationMinutes)
	assert.Equal(t, "Appointment booked successfully", out.Message)
}

// TestBookAppointmentUseCase_NoAvailability_SlotAlreadyBooked books the only
// (technician, bay) pair at 10:00, then verifies a second request for the same
// slot returns ErrNoAvailability (FindAvailableTechnicianAndBay returns nil).
func TestBookAppointmentUseCase_NoAvailability_SlotAlreadyBooked(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	uc := newBookingUC(t)
	start := slotAt(10)

	_, err := uc.BookAppointment(context.Background(), bookingInput(0, start))
	require.NoError(t, err, "first booking should succeed")

	_, err = uc.BookAppointment(context.Background(), bookingInput(1, start))
	assert.ErrorIs(t, err, usecase.ErrNoAvailability,
		"second booking for the same slot should return ErrNoAvailability")
}

// TestBookAppointmentUseCase_OverlappingSlot_ReturnsNoAvailability books 10:00–10:30,
// then verifies an overlapping 10:15–10:45 request is also rejected.
func TestBookAppointmentUseCase_OverlappingSlot_ReturnsNoAvailability(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	uc := newBookingUC(t)
	start := slotAt(10)

	_, err := uc.BookAppointment(context.Background(), bookingInput(0, start))
	require.NoError(t, err, "first booking should succeed")

	// 10:15 overlaps with the existing 10:00-10:30 appointment
	overlap := start.Add(15 * time.Minute)
	_, err = uc.BookAppointment(context.Background(), bookingInput(1, overlap))
	assert.ErrorIs(t, err, usecase.ErrNoAvailability,
		"overlapping slot should return ErrNoAvailability")
}

// TestBookAppointmentUseCase_Concurrent_TenWorkers_ExactlyOneSucceeds fires 10
// goroutines simultaneously at the same slot against capacity=1 (dealership1).
// The Redis lock serialises the requests: exactly one acquires the lock and
// inserts; the others receive ErrNoAvailability or ErrResourceLocked.
func TestBookAppointmentUseCase_Concurrent_TenWorkers_ExactlyOneSucceeds(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	uc := newBookingUC(t)
	start := slotAt(11)

	type result struct {
		out *usecase.AppoinmentBookingOutput
		err error
	}
	results := make([]result, nWorkers)

	var ready, gate, done sync.WaitGroup
	ready.Add(nWorkers)
	gate.Add(1)
	done.Add(nWorkers)

	for i := range nWorkers {
		go func(i int) {
			defer done.Done()
			ready.Done()
			gate.Wait()
			out, err := uc.BookAppointment(context.Background(), bookingInput(i, start))
			results[i] = result{out, err}
		}(i)
	}

	ready.Wait()
	gate.Done()
	done.Wait()

	successes := 0
	for i, r := range results {
		if r.err == nil {
			successes++
		} else if !errors.Is(r.err, usecase.ErrNoAvailability) && !errors.Is(r.err, usecase.ErrResourceLocked) {
			t.Errorf("worker %d: unexpected error: %v", i, r.err)
		}
	}
	if successes != 1 {
		t.Errorf("capacity=1, %d workers: expected exactly 1 success, got %d", nWorkers, successes)
	}
}

// TestBookAppointmentUseCase_CapacityTwo_TwoWorkers_BothSucceed fires two
// goroutines simultaneously against dealership2 (2 bays, 2 technicians).
// Both must eventually succeed.
//
// Why retries are needed here: FindAvailableTechnicianAndBay is deterministic
// (ORDER BY name, bay_number LIMIT 1), so both goroutines select the same
// "first" pair before either has committed. Worker 0 wins the Redis lock;
// worker 1 receives ErrResourceLocked — the signal to retry. On the next
// attempt the committed appointment excludes the first pair, and worker 1
// is assigned the second (tech, bay) pair.
func TestBookAppointmentUseCase_CapacityTwo_TwoWorkers_BothSucceed(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	uc := newBookingUC(t)
	start := slotAt(12)

	type result struct {
		out *usecase.AppoinmentBookingOutput
		err error
	}
	results := make([]result, 2)

	var ready, gate, done sync.WaitGroup
	ready.Add(2)
	gate.Add(1)
	done.Add(2)

	for i := range 2 {
		go func(i int) {
			defer done.Done()
			ready.Done()
			gate.Wait()

			input := &usecase.AppoinmentBookingInput{
				CustomerID:       fix.customers[i],
				DealershipID:     fix.dealership2,
				VehicleID:        fix.vehicles[i],
				Services:         []string{fix.serviceID},
				DesiredStartTime: start,
			}
			// ErrResourceLocked is a transient "retry" signal: another request
			// is mid-commit on the same pair. Back off briefly and retry so the
			// committed appointment is visible and FindAvailableTechnicianAndBay
			// can return the second free pair.
			var out *usecase.AppoinmentBookingOutput
			var bookErr error
			for attempt := 0; attempt < 10; attempt++ {
				out, bookErr = uc.BookAppointment(context.Background(), input)
				if !errors.Is(bookErr, usecase.ErrResourceLocked) {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			results[i] = result{out, bookErr}
		}(i)
	}

	ready.Wait()
	gate.Done()
	done.Wait()

	for i, r := range results {
		require.NoErrorf(t, r.err, "worker %d: expected success with capacity=2 (after retries)", i)
		assert.NotEmpty(t, results[i].out.AppointmentID)
	}
	assert.NotEqual(t, results[0].out.AppointmentID, results[1].out.AppointmentID,
		"both workers should receive distinct appointment IDs")
}
