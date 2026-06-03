package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"keyloop-test/internal/domain"
	"keyloop-test/internal/usecase"
)

// ── spy types ─────────────────────────────────────────────────────────────────
// These extend the shared stubs (defined in book_appointment_test.go) with call
// counters so deferred ReleaseLock / AcquireLock behaviour can be asserted.

type spyBookingRepo struct {
	resources    *domain.AvailableResources
	availErr     error
	insertErr    error
	insertCalled bool
	lastInserted *domain.Appointment
}

func (r *spyBookingRepo) FindAvailableTechnicianAndBay(_ context.Context, _ string, _, _ time.Time, _ []string) (*domain.AvailableResources, error) {
	return r.resources, r.availErr
}

func (r *spyBookingRepo) InsertAppointment(_ context.Context, appt *domain.Appointment) (*domain.Appointment, error) {
	r.insertCalled = true
	r.lastInserted = appt
	if r.insertErr != nil {
		return nil, r.insertErr
	}
	out := *appt
	out.ID = "appt-book-001"
	return &out, nil
}

type spyLocker struct {
	acquired     bool
	acquireErr   error
	acquireCalls int
	releaseCalls int
}

func (l *spyLocker) AcquireLock(_ context.Context, _, _ string, _, _ time.Time) (bool, error) {
	l.acquireCalls++
	return l.acquired, l.acquireErr
}

func (l *spyLocker) ReleaseLock(_ context.Context, _, _ string, _, _ time.Time) error {
	l.releaseCalls++
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func validBookingInput() *usecase.AppoinmentBookingInput {
	return &usecase.AppoinmentBookingInput{
		CustomerID:       "c-001",
		DealershipID:     "d-001",
		VehicleID:        "v-001",
		Services:         []string{"svc-oil"},
		DesiredStartTime: futureTime(9), // defined in book_appointment_test.go
	}
}

// newBookingUC wires a BookAppointment use case with always-happy background
// stubs and the provided spy repo/locker so tests can assert on call counts.
func newBookingUC(repo *spyBookingRepo, locker *spyLocker) *usecase.AppoinmentBookingUseCase {
	return usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{dealership: openDealership()},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		&stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001", CustomerID: "c-001"}},
		&stubCache{
			service: &domain.Service{ID: "svc-oil", Name: "Oil Change", EstimatedMinutes: 30, Price: 29.99},
			hit:     true,
		},
		&stubAvailabilityRepo{},
		&stubAppointmentRepo{},
		repo,
		locker,
		&stubTracer{},
	)
}

// ── tests ─────────────────────────────────────────────────────────────────────

// (a) Happy path: resources found, lock acquired, appointment inserted, lock released.
func TestBookAppointment_HappyPath_LockAcquiredAndReleased(t *testing.T) {
	repo := &spyBookingRepo{
		resources: &domain.AvailableResources{TechnicianID: "tech-1", BayID: "bay-1"},
	}
	locker := &spyLocker{acquired: true}

	out, err := newBookingUC(repo, locker).BookAppointment(context.Background(), validBookingInput())

	require.NoError(t, err)
	assert.Equal(t, "appt-book-001", out.AppointmentID)
	assert.Equal(t, 30, out.DurationMinutes)
	assert.True(t, repo.insertCalled, "InsertAppointment should be called")
	assert.Equal(t, 1, locker.releaseCalls, "ReleaseLock should be deferred exactly once")
}

// (b) FindAvailableTechnicianAndBay returns nil → ErrNoAvailability, AcquireLock never called.
func TestBookAppointment_NoAvailability_ReturnsErrNoAvailability(t *testing.T) {
	repo := &spyBookingRepo{resources: nil}
	locker := &spyLocker{}

	_, err := newBookingUC(repo, locker).BookAppointment(context.Background(), validBookingInput())

	assert.ErrorIs(t, err, usecase.ErrNoAvailability)
	assert.Equal(t, 0, locker.acquireCalls, "AcquireLock must not be called when no resources found")
}

// (c) AcquireLock returns (false, nil) → ErrResourceLocked; InsertAppointment and ReleaseLock not called.
func TestBookAppointment_LockAlreadyTaken_ReturnsErrResourceLocked(t *testing.T) {
	repo := &spyBookingRepo{
		resources: &domain.AvailableResources{TechnicianID: "tech-1", BayID: "bay-1"},
	}
	locker := &spyLocker{acquired: false}

	_, err := newBookingUC(repo, locker).BookAppointment(context.Background(), validBookingInput())

	assert.ErrorIs(t, err, usecase.ErrResourceLocked)
	assert.False(t, repo.insertCalled, "InsertAppointment must not be called when lock not acquired")
	assert.Equal(t, 0, locker.releaseCalls, "ReleaseLock must not be deferred when lock was never acquired")
}

// (d) InsertAppointment returns ErrDuplicateBooking → ErrNoAvailability; lock is still released.
func TestBookAppointment_DBDuplicateConstraint_ReturnsErrNoAvailability(t *testing.T) {
	repo := &spyBookingRepo{
		resources: &domain.AvailableResources{TechnicianID: "tech-1", BayID: "bay-1"},
		insertErr: domain.ErrDuplicateBooking,
	}
	locker := &spyLocker{acquired: true}

	_, err := newBookingUC(repo, locker).BookAppointment(context.Background(), validBookingInput())

	assert.ErrorIs(t, err, usecase.ErrNoAvailability)
	assert.Equal(t, 1, locker.releaseCalls, "ReleaseLock must be deferred even when DB returns a duplicate constraint")
}

// (e) InsertAppointment returns a generic DB error → original error reachable via errors.Is; lock still released.
func TestBookAppointment_DBGenericError_LockReleased(t *testing.T) {
	dbErr := errors.New("connection reset by peer")
	repo := &spyBookingRepo{
		resources: &domain.AvailableResources{TechnicianID: "tech-1", BayID: "bay-1"},
		insertErr: dbErr,
	}
	locker := &spyLocker{acquired: true}

	_, err := newBookingUC(repo, locker).BookAppointment(context.Background(), validBookingInput())

	assert.ErrorIs(t, err, dbErr, "original DB error must be reachable through the %%w wrap")
	assert.Equal(t, 1, locker.releaseCalls, "ReleaseLock must be deferred even when DB returns a generic error")
}
