package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"keyloop-test/internal/domain"
	"keyloop-test/internal/port"
	"keyloop-test/internal/usecase"
)

// ─── stubs ───────────────────────────────────────────────────────────────────

type stubDealershipRepo struct {
	dealership *domain.Dealership
	err        error
}

func (s *stubDealershipRepo) GetDealership(_ context.Context, _ string) (*domain.Dealership, error) {
	return s.dealership, s.err
}
func (s *stubDealershipRepo) ListDealerships(_ context.Context) ([]*domain.Dealership, error) {
	return nil, nil
}
func (s *stubDealershipRepo) CreateDealership(_ context.Context, _ *domain.Dealership) error {
	return nil
}
func (s *stubDealershipRepo) UpdateDealership(_ context.Context, _ *domain.Dealership) error {
	return nil
}
func (s *stubDealershipRepo) DeleteDealership(_ context.Context, _ string) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type stubServiceBayRepo struct{}

func (s *stubServiceBayRepo) GetServiceBay(_ context.Context, _ string) (*domain.ServiceBay, error) {
	return nil, nil
}
func (s *stubServiceBayRepo) ListServiceBaysByDealership(_ context.Context, _ string) ([]*domain.ServiceBay, error) {
	return nil, nil
}
func (s *stubServiceBayRepo) GetAvailableServiceBays(_ context.Context, _ string, _, _ time.Time) ([]*domain.ServiceBay, error) {
	return nil, nil
}
func (s *stubServiceBayRepo) CreateServiceBay(_ context.Context, _ *domain.ServiceBay) error {
	return nil
}
func (s *stubServiceBayRepo) UpdateServiceBay(_ context.Context, _ *domain.ServiceBay) error {
	return nil
}
func (s *stubServiceBayRepo) DeleteServiceBay(_ context.Context, _ string) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type stubTechnicianRepo struct{}

func (s *stubTechnicianRepo) GetTechnician(_ context.Context, _ string) (*domain.Technician, error) {
	return nil, nil
}
func (s *stubTechnicianRepo) ListTechniciansByDealership(_ context.Context, _ string) ([]*domain.Technician, error) {
	return nil, nil
}
func (s *stubTechnicianRepo) GetAvailableTechnicians(_ context.Context, _ string, _, _ time.Time, _ []string) ([]*domain.Technician, error) {
	return nil, nil
}
func (s *stubTechnicianRepo) CreateTechnician(_ context.Context, _ *domain.Technician) error {
	return nil
}
func (s *stubTechnicianRepo) UpdateTechnician(_ context.Context, _ *domain.Technician) error {
	return nil
}
func (s *stubTechnicianRepo) DeleteTechnician(_ context.Context, _ string) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type stubServiceDefRepo struct {
	service *domain.Service
	err     error
}

func (s *stubServiceDefRepo) GetService(_ context.Context, _ string) (*domain.Service, error) {
	return s.service, s.err
}
func (s *stubServiceDefRepo) ListServices(_ context.Context) ([]*domain.Service, error) {
	return nil, nil
}
func (s *stubServiceDefRepo) CreateService(_ context.Context, _ *domain.Service) error { return nil }
func (s *stubServiceDefRepo) UpdateService(_ context.Context, _ *domain.Service) error { return nil }
func (s *stubServiceDefRepo) DeleteService(_ context.Context, _ string) error          { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type stubVehicleRepo struct {
	vehicle *domain.Vehicle
	err     error
}

func (s *stubVehicleRepo) GetVehicle(_ context.Context, _ string) (*domain.Vehicle, error) {
	return s.vehicle, s.err
}
func (s *stubVehicleRepo) ListVehiclesByCustomer(_ context.Context, _ string) ([]domain.Vehicle, error) {
	return nil, nil
}
func (s *stubVehicleRepo) CreateVehicle(_ context.Context, _ *domain.Vehicle) error { return nil }
func (s *stubVehicleRepo) UpdateVehicle(_ context.Context, _ *domain.Vehicle) error { return nil }
func (s *stubVehicleRepo) DeleteVehicle(_ context.Context, _ string) error          { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type stubAvailabilityRepo struct{}

func (s *stubAvailabilityRepo) GetAvailableSlots(_ context.Context, _, _ time.Time, _ string, _ time.Duration, _ []string) ([]domain.AvailableSlot, error) {
	return nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type stubAppointmentRepo struct {
	appt *domain.Appointment
	err  error
	// capture the last call for assertion
	lastCreated *domain.Appointment
}

func (s *stubAppointmentRepo) CreateAppointment(_ context.Context, appt *domain.Appointment) (*domain.Appointment, error) {
	s.lastCreated = appt
	if s.err != nil {
		return nil, s.err
	}
	out := *appt
	out.ID = "appt-001"
	return &out, nil
}
func (s *stubAppointmentRepo) GetAppointment(_ context.Context, _ string) (*domain.Appointment, error) {
	return s.appt, s.err
}
func (s *stubAppointmentRepo) ListAppointmentsByDealership(_ context.Context, _ string) ([]domain.Appointment, error) {
	return nil, nil
}
func (s *stubAppointmentRepo) ListAppointmentsByCustomer(_ context.Context, _ string) ([]domain.Appointment, error) {
	return nil, nil
}
func (s *stubAppointmentRepo) ListAppointmentsByCustomerAndDealership(_ context.Context, _, _ string) ([]domain.Appointment, error) {
	return nil, nil
}
func (s *stubAppointmentRepo) UpdateAppointmentStatus(_ context.Context, _ *domain.Appointment, _ domain.AppointmentStatus) error {
	return nil
}
func (s *stubAppointmentRepo) UpdateAppointment(_ context.Context, _ string, _ domain.AppointmentStatus, _ string) (*domain.Appointment, error) {
	return nil, nil
}
func (s *stubAppointmentRepo) DeleteAppointment(_ context.Context, _ string) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────

type stubCache struct {
	service  *domain.Service
	hit      bool
	err      error
	setCalls int
}

func (s *stubCache) GetService(_ context.Context, _ string) (*domain.Service, bool, error) {
	return s.service, s.hit, s.err
}
func (s *stubCache) SetService(_ context.Context, _ string, _ *domain.Service, _ time.Duration) error {
	s.setCalls++
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Booking-path stubs (used by tests that exercise FindAvailableTechnicianAndBay
// + AcquireLock + InsertAppointment).

type stubTracer struct{}

func (t *stubTracer) Start(ctx context.Context, _ string, _ ...port.SpanAttr) (context.Context, port.Span) {
	return ctx, &stubSpan{}
}

type stubSpan struct{}

func (s *stubSpan) End()                    {}
func (s *stubSpan) RecordError(_ error)     {}
func (s *stubSpan) SetErrorStatus(_ string) {}

type stubBookingRepo struct {
	resources    *domain.AvailableResources
	availErr     error
	insertErr    error
	lastInserted *domain.Appointment
}

func (r *stubBookingRepo) FindAvailableTechnicianAndBay(_ context.Context, _ string, _, _ time.Time, _ []string) (*domain.AvailableResources, error) {
	return r.resources, r.availErr
}

func (r *stubBookingRepo) InsertAppointment(_ context.Context, appt *domain.Appointment) (*domain.Appointment, error) {
	r.lastInserted = appt
	if r.insertErr != nil {
		return nil, r.insertErr
	}
	out := *appt
	out.ID = "appt-001"
	return &out, nil
}

type stubBookingLocker struct {
	acquired     bool
	releaseCalls int
}

func (l *stubBookingLocker) AcquireLock(_ context.Context, _, _ string, _, _ time.Time) (bool, error) {
	return l.acquired, nil
}
func (l *stubBookingLocker) ReleaseLock(_ context.Context, _, _ string, _, _ time.Time) error {
	l.releaseCalls++
	return nil
}

// defaultBookingRepo returns a repo that always finds one available pair.
func defaultBookingRepo() *stubBookingRepo {
	return &stubBookingRepo{
		resources: &domain.AvailableResources{TechnicianID: "tech-1", BayID: "bay-1"},
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// dealership that is open 08:00–18:00 UTC
func openDealership() *domain.Dealership {
	return &domain.Dealership{
		ID:          "d-001",
		IsActive:    true,
		OpenTime:    8 * time.Hour,
		CloseTime:   18 * time.Hour,
		WorkingDays: []time.Weekday{0, 1, 2, 3, 4, 5, 6}, // all days — generic test fixture
	}
}

func weekdayDealership() *domain.Dealership {
	return &domain.Dealership{
		ID:          "d-001",
		IsActive:    true,
		OpenTime:    8 * time.Hour,
		CloseTime:   18 * time.Hour,
		WorkingDays: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
	}
}

// nextWeekend returns a future Saturday at the given hour.
func nextWeekend(hour int) time.Time {
	t := time.Now().UTC()
	for t.Weekday() != time.Saturday {
		t = t.Add(24 * time.Hour)
	}
	t = t.Add(24 * time.Hour) // ensure it's in the future even on Saturdays
	return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, time.UTC)
}

// futureTime returns a time in UTC at the given hour of day, tomorrow.
// Using UTC ensures timeOfDay comparisons in domain logic are timezone-agnostic.
func futureTime(hour int) time.Time {
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, 0, 0, 0, time.UTC)
}

func oilChangeService() *domain.Service {
	return &domain.Service{
		ID:               "svc-oil",
		Name:             "Oil Change",
		EstimatedMinutes: 30,
		Price:            29.99,
	}
}

func tireRotationService() *domain.Service {
	return &domain.Service{
		ID:               "svc-tire",
		Name:             "Tire Rotation",
		EstimatedMinutes: 45,
		Price:            19.99,
	}
}

func newUC(
	dealershipRepo *stubDealershipRepo,
	serviceDefRepo *stubServiceDefRepo,
	vehicleRepo *stubVehicleRepo,
	apptRepo *stubAppointmentRepo,
	cache *stubCache,
) *usecase.AppoinmentBookingUseCase {
	return usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		serviceDefRepo,
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		apptRepo,
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestBookAppointment_ServiceNotFound_CacheMissAndDBMiss(t *testing.T) {
	cache := &stubCache{hit: false}
	serviceRepo := &stubServiceDefRepo{err: errors.New("not found")}
	uc := newUC(&stubDealershipRepo{}, serviceRepo, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"unknown_service"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBookAppointment_DesiredTimeInPast(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	uc := newUC(&stubDealershipRepo{}, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: time.Now().Add(-1 * time.Hour), // past
		VehicleID:        "v-001",
	})
	if !errors.Is(err, usecase.ErrDesiredDateInPast) {
		t.Errorf("want ErrDesiredDateInPast, got %v", err)
	}
}

func TestBookAppointment_DealershipNotFound(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{err: errors.New("not found")}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-999",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if !errors.Is(err, usecase.ErrDealershipNotFound) {
		t.Errorf("want ErrDealershipNotFound, got %v", err)
	}
}

func TestBookAppointment_SlotStartsBeforeOpen(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	// dealership opens at 08:00; slot at 07:30 is before open
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(7).Add(30 * time.Minute), // 07:30
		VehicleID:        "v-001",
	})
	if err == nil {
		t.Fatal("expected error for slot before open, got nil")
	}
}

func TestBookAppointment_SlotEndsAfterClose(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	// dealership closes at 18:00; a 30-min service starting at 17:45 ends at 18:15
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(17).Add(45 * time.Minute), // 17:45 + 30 min = 18:15 > close
		VehicleID:        "v-001",
	})
	if err == nil {
		t.Fatal("expected error for slot ending after close, got nil")
	}
}

// ─── duration / close-time boundary tests ────────────────────────────────────

// TestBookAppointment_SlotEndsExactlyAtClose_Succeeds checks the boundary:
// IsSlotWithinHours uses <=, so an appointment that ends exactly at close must
// be accepted.
func TestBookAppointment_SlotEndsExactlyAtClose_Succeeds(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	apptRepo := &stubAppointmentRepo{}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, vehicleRepo, apptRepo, cache)

	// 17:30 + 30 min = 18:00 exactly — should succeed
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(17).Add(30 * time.Minute),
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Errorf("expected success for slot ending exactly at close, got %v", err)
	}
}

// TestBookAppointment_SlotEndsOneMinuteAfterClose_Fails confirms that even a
// one-minute overrun past closing is rejected.
func TestBookAppointment_SlotEndsOneMinuteAfterClose_Fails(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	// 17:31 + 30 min = 18:01 — just one minute past close
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(17).Add(31 * time.Minute),
		VehicleID:        "v-001",
	})
	if err == nil {
		t.Fatal("expected error for slot ending 1 minute after close, got nil")
	}
}

// TestBookAppointment_CombinedServiceDurationExceedsClose_Fails ensures that
// multi-service bookings are also checked: each individual service might fit,
// but the combined end time must still fall within hours.
// oil_change(30min) + tire_rotation(45min) = 75min; starting at 17:00 → 18:15 > close.
func TestBookAppointment_CombinedServiceDurationExceedsClose_Fails(t *testing.T) {
	cache := &stubCache{hit: false}
	services := map[string]*domain.Service{
		"oil_change":    oilChangeService(),
		"tire_rotation": tireRotationService(),
	}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&multiServiceDefRepo{services: services},
		&stubVehicleRepo{},
		cache,
		&stubAvailabilityRepo{},
		&stubAppointmentRepo{},
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	// 17:00 + 75min = 18:15 — exceeds close even though 17:00 is within hours
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change", "tire_rotation"},
		DesiredStartTime: futureTime(17),
		VehicleID:        "v-001",
	})
	if err == nil {
		t.Fatal("expected error for combined duration exceeding close time, got nil")
	}
}

// TestBookAppointment_CombinedServiceEndsExactlyAtClose_Succeeds is the mirror
// of the above: 75min starting at 16:45 ends at exactly 18:00 and must succeed.
func TestBookAppointment_CombinedServiceEndsExactlyAtClose_Succeeds(t *testing.T) {
	cache := &stubCache{hit: false}
	services := map[string]*domain.Service{
		"oil_change":    oilChangeService(),    // 30 min
		"tire_rotation": tireRotationService(), // 45 min → total 75 min
	}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	apptRepo := &stubAppointmentRepo{}
	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&multiServiceDefRepo{services: services},
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		apptRepo,
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	// 16:45 + 75min = 18:00 exactly
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change", "tire_rotation"},
		DesiredStartTime: futureTime(16).Add(45 * time.Minute),
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Errorf("expected success for combined duration ending exactly at close, got %v", err)
	}
}

// TestBookAppointment_LongSingleServiceExceedsClose_Fails tests a single
// service whose duration alone (3 hours) pushes the appointment past closing
// when started late in the day.
func TestBookAppointment_LongSingleServiceExceedsClose_Fails(t *testing.T) {
	longService := &domain.Service{ID: "svc-engine", Name: "Engine Overhaul", EstimatedMinutes: 180}
	cache := &stubCache{service: longService, hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, cache)

	// 16:00 + 180min = 19:00 — way past close
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"engine_overhaul"},
		DesiredStartTime: futureTime(16),
		VehicleID:        "v-001",
	})
	if err == nil {
		t.Fatal("expected error for long service exceeding close time, got nil")
	}
}

// TestBookAppointment_LongSingleServiceEndsExactlyAtClose_Succeeds verifies
// that the same long service is accepted when it starts exactly at open and
// finishes exactly at close (10 hours, 08:00–18:00).
func TestBookAppointment_LongSingleServiceEndsExactlyAtClose_Succeeds(t *testing.T) {
	fullDayService := &domain.Service{ID: "svc-fullday", Name: "Full Day Service", EstimatedMinutes: 600}
	cache := &stubCache{service: fullDayService, hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	apptRepo := &stubAppointmentRepo{}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, vehicleRepo, apptRepo, cache)

	// 08:00 + 600min = 18:00 exactly — should succeed
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"full_day_service"},
		DesiredStartTime: futureTime(8),
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Errorf("expected success for full-day service ending exactly at close, got %v", err)
	}
}

func TestBookAppointment_VehicleNotOwnedByCustomer(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	// vehicle exists but belongs to a different customer
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001", CustomerID: "c-OTHER"}}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, vehicleRepo, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		CustomerID:       "c-001",
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if !errors.Is(err, usecase.ErrVehicleNotOwnedByCustomer) {
		t.Errorf("want ErrVehicleNotOwnedByCustomer, got %v", err)
	}
}

func TestBookAppointment_VehicleNotFound(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{err: errors.New("vehicle not found")}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, vehicleRepo, &stubAppointmentRepo{}, cache)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-999",
	})
	if !errors.Is(err, usecase.ErrVehicleNotFound) {
		t.Errorf("want ErrVehicleNotFound, got %v", err)
	}
}

func TestBookAppointment_RepoError(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	repoErr := errors.New("no available bay or technician")
	bookingRepo := &stubBookingRepo{
		resources: &domain.AvailableResources{TechnicianID: "tech-1", BayID: "bay-1"},
		insertErr: repoErr,
	}
	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		&stubAppointmentRepo{},
		bookingRepo,
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("want repo error propagated, got %v", err)
	}
}

func TestBookAppointment_HappyPath_CacheHit(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	apptRepo := &stubAppointmentRepo{}
	uc := newUC(dealershipRepo, &stubServiceDefRepo{}, vehicleRepo, apptRepo, cache)

	start := futureTime(9)
	out, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: start,
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AppointmentID == "" {
		t.Error("expected non-empty AppointmentID")
	}
	if out.DurationMinutes != 30 {
		t.Errorf("DurationMinutes = %d, want 30", out.DurationMinutes)
	}
	if out.Message == "" {
		t.Error("expected non-empty Message")
	}
	// cache hit means SetService should NOT have been called
	if cache.setCalls != 0 {
		t.Errorf("SetService called %d times on cache hit, want 0", cache.setCalls)
	}
}

func TestBookAppointment_HappyPath_CacheMissFallsBackToRepo(t *testing.T) {
	cache := &stubCache{hit: false}
	serviceRepo := &stubServiceDefRepo{service: oilChangeService()}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	apptRepo := &stubAppointmentRepo{}
	uc := newUC(dealershipRepo, serviceRepo, vehicleRepo, apptRepo, cache)

	out, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.DurationMinutes != 30 {
		t.Errorf("DurationMinutes = %d, want 30", out.DurationMinutes)
	}
	// cache miss means SetService should have been called once to warm the cache
	if cache.setCalls != 1 {
		t.Errorf("SetService called %d times on cache miss, want 1", cache.setCalls)
	}
}

func TestBookAppointment_MultipleServices_DurationSums(t *testing.T) {
	// oil_change=30min, tire_rotation=45min → total 75min
	callCount := 0
	cache := &stubCache{}
	cache.hit = false

	services := map[string]*domain.Service{
		"oil_change":    oilChangeService(),
		"tire_rotation": tireRotationService(),
	}
	serviceRepo := &stubServiceDefRepo{}
	// override with a closure-based stub via a custom type
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	apptRepo := &stubAppointmentRepo{}

	// Use a custom serviceRepo that returns different services by ID
	customServiceRepo := &multiServiceDefRepo{services: services}
	_ = callCount

	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		customServiceRepo,
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		apptRepo,
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)
	_ = serviceRepo

	out, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change", "tire_rotation"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.DurationMinutes != 75 {
		t.Errorf("DurationMinutes = %d, want 75 (30+45)", out.DurationMinutes)
	}
}

func TestBookAppointment_NotesPassedThrough(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	bookingRepo := defaultBookingRepo()
	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		&stubAppointmentRepo{},
		bookingRepo,
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	const note = "please check the brakes too"
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
		Notes:            note,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bookingRepo.lastInserted == nil {
		t.Fatal("expected InsertAppointment to be called")
	}
	if bookingRepo.lastInserted.Notes != note {
		t.Errorf("Notes = %q, want %q", bookingRepo.lastInserted.Notes, note)
	}
}

func TestBookAppointment_StatusIsPendingOnCreation(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	bookingRepo := defaultBookingRepo()
	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		&stubAppointmentRepo{},
		bookingRepo,
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: futureTime(9),
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bookingRepo.lastInserted.Status != domain.AppointmentStatusPending {
		t.Errorf("Status = %v, want pending", bookingRepo.lastInserted.Status)
	}
}

func TestBookAppointment_EndTimeIsStartPlusDuration(t *testing.T) {
	cache := &stubCache{service: oilChangeService(), hit: true}
	dealershipRepo := &stubDealershipRepo{dealership: openDealership()}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	bookingRepo := defaultBookingRepo()
	uc := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		cache,
		&stubAvailabilityRepo{},
		&stubAppointmentRepo{},
		bookingRepo,
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	start := futureTime(9)
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		DealershipID:     "d-001",
		Services:         []string{"oil_change"},
		DesiredStartTime: start,
		VehicleID:        "v-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantEnd := start.Add(30 * time.Minute)
	if !bookingRepo.lastInserted.EndTime.Equal(wantEnd) {
		t.Errorf("EndTime = %v, want %v", bookingRepo.lastInserted.EndTime, wantEnd)
	}
}

// configuredAvailabilityRepo returns a fixed set of slots for AvailableSlots unit tests.
type configuredAvailabilityRepo struct {
	slots []domain.AvailableSlot
	err   error
}

func (s *configuredAvailabilityRepo) GetAvailableSlots(_ context.Context, _, _ time.Time, _ string, _ time.Duration, _ []string) ([]domain.AvailableSlot, error) {
	return s.slots, s.err
}

// ─── AvailableSlots usecase unit tests ───────────────────────────────────────

func availableSlotsReq() *usecase.AvailableSlotsInput {
	return &usecase.AvailableSlotsInput{
		DealershipID: "d-001",
		Services:     []string{"oil_change"},
		VehicleID:    "v-001",
		DesiredDate:  futureTime(8),
	}
}

func TestAvailableSlots_DesiredDateInPast(t *testing.T) {
	uc := newUC(&stubDealershipRepo{}, &stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{}, &stubCache{})
	req := availableSlotsReq()
	req.DesiredDate = time.Now().Add(-1 * time.Hour)
	_, err := uc.AvailableSlots(context.Background(), req)
	if !errors.Is(err, usecase.ErrDesiredDateInPast) {
		t.Errorf("want ErrDesiredDateInPast, got %v", err)
	}
}

func TestAvailableSlots_DealershipNotFound(t *testing.T) {
	uc := newUC(
		&stubDealershipRepo{err: errors.New("not found")},
		&stubServiceDefRepo{}, &stubVehicleRepo{}, &stubAppointmentRepo{},
		&stubCache{service: oilChangeService(), hit: true},
	)
	_, err := uc.AvailableSlots(context.Background(), availableSlotsReq())
	if !errors.Is(err, usecase.ErrDealershipNotFound) {
		t.Errorf("want ErrDealershipNotFound, got %v", err)
	}
}

func TestAvailableSlots_VehicleNotOwnedByCustomer(t *testing.T) {
	// vehicle exists but belongs to a different customer
	uc := newUC(
		&stubDealershipRepo{dealership: openDealership()},
		&stubServiceDefRepo{},
		&stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001", CustomerID: "c-OTHER"}},
		&stubAppointmentRepo{},
		&stubCache{service: oilChangeService(), hit: true},
	)
	req := availableSlotsReq()
	req.CustomerID = "c-001"
	_, err := uc.AvailableSlots(context.Background(), req)
	if !errors.Is(err, usecase.ErrVehicleNotOwnedByCustomer) {
		t.Errorf("want ErrVehicleNotOwnedByCustomer, got %v", err)
	}
}

func TestAvailableSlots_VehicleNotFound(t *testing.T) {
	uc := newUC(
		&stubDealershipRepo{dealership: openDealership()},
		&stubServiceDefRepo{},
		&stubVehicleRepo{err: errors.New("vehicle not found")},
		&stubAppointmentRepo{},
		&stubCache{service: oilChangeService(), hit: true},
	)
	_, err := uc.AvailableSlots(context.Background(), availableSlotsReq())
	if !errors.Is(err, usecase.ErrVehicleNotFound) {
		t.Errorf("want ErrVehicleNotFound, got %v", err)
	}
}

// TestAvailableSlots_FiltersOutsideBusinessHours confirms that the usecase
// removes raw slots whose [start, start+duration) window falls outside the
// dealership's open/close hours (08:00–18:00).
func TestAvailableSlots_FiltersOutsideBusinessHours(t *testing.T) {
	// Raw slots returned by the slot repo:
	//   07:30 → [07:30, 08:00) — start 07:30 < 08:00 open → filtered
	//   09:00 → [09:00, 09:30) — within hours → kept
	//   17:30 → [17:30, 18:00) — ends exactly at close → kept
	//   18:00 → [18:00, 18:30) — end 18:30 > 18:00 close → filtered
	rawSlots := []domain.AvailableSlot{
		{Start: futureTime(7).Add(30 * time.Minute), End: futureTime(8)},
		{Start: futureTime(9), End: futureTime(9).Add(30 * time.Minute)},
		{Start: futureTime(17).Add(30 * time.Minute), End: futureTime(18)},
		{Start: futureTime(18), End: futureTime(18).Add(30 * time.Minute)},
	}
	slotRepo := &configuredAvailabilityRepo{slots: rawSlots}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}

	uc := usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{dealership: openDealership()},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		&stubCache{service: oilChangeService(), hit: true},
		slotRepo,
		&stubAppointmentRepo{},
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	out, err := uc.AvailableSlots(context.Background(), availableSlotsReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.AvailableSlots) != 2 {
		t.Errorf("want 2 slots after filtering, got %d", len(out.AvailableSlots))
	}
}

// TestAvailableSlots_EmptyResult_ReturnsEmptySlice verifies that when the repo
// returns no slots, the usecase returns an empty (non-nil) slice.
func TestAvailableSlots_EmptyResult_ReturnsEmptySlice(t *testing.T) {
	slotRepo := &configuredAvailabilityRepo{slots: []domain.AvailableSlot{}}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}

	uc := usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{dealership: openDealership()},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		&stubCache{service: oilChangeService(), hit: true},
		slotRepo,
		&stubAppointmentRepo{},
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	out, err := uc.AvailableSlots(context.Background(), availableSlotsReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AvailableSlots == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(out.AvailableSlots) != 0 {
		t.Errorf("want 0 slots, got %d", len(out.AvailableSlots))
	}
}

func TestBookAppointment_WeekendRejected(t *testing.T) {
	uc := newUC(
		&stubDealershipRepo{dealership: weekdayDealership()},
		&stubServiceDefRepo{},
		&stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}},
		&stubAppointmentRepo{},
		&stubCache{service: oilChangeService(), hit: true},
	)
	_, err := uc.BookAppointment(context.Background(), &usecase.AppoinmentBookingInput{
		CustomerID:       "c-001",
		DealershipID:     "d-001",
		VehicleID:        "v-001",
		Services:         []string{"svc-oil"},
		DesiredStartTime: nextWeekend(10),
	})
	if !errors.Is(err, usecase.ErrSlotNotOnWorkingDay) {
		t.Errorf("want ErrSlotNotOnWorkingDay, got %v", err)
	}
}

// TestAvailableSlots_SlotRepoError verifies that a DB error from GetAvailableSlots
// is propagated to the caller.
func TestAvailableSlots_SlotRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	slotRepo := &configuredAvailabilityRepo{err: repoErr}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}

	uc := usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{dealership: openDealership()},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		&stubCache{service: oilChangeService(), hit: true},
		slotRepo,
		&stubAppointmentRepo{},
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	_, err := uc.AvailableSlots(context.Background(), availableSlotsReq())
	if !errors.Is(err, repoErr) {
		t.Errorf("want slot repo error propagated, got %v", err)
	}
}

// TestAvailableSlots_FiltersNonWorkingDaySlots confirms that slots whose start
// falls on a non-working day are removed by the per-slot filter even if they
// satisfy the time-of-day constraint (the IsWorkingDay check in the filter loop).
func TestAvailableSlots_FiltersNonWorkingDaySlots(t *testing.T) {
	// weekdayDealership only allows Mon–Fri
	monday := time.Now().UTC()
	for monday.Weekday() != time.Monday {
		monday = monday.Add(24 * time.Hour)
	}
	monday = monday.Add(24 * time.Hour) // ensure it's in the future even when today is Monday
	saturday := monday.Add(5 * 24 * time.Hour)

	mondaySlot := domain.AvailableSlot{
		Start: time.Date(monday.Year(), monday.Month(), monday.Day(), 9, 0, 0, 0, time.UTC),
		End:   time.Date(monday.Year(), monday.Month(), monday.Day(), 9, 30, 0, 0, time.UTC),
	}
	saturdaySlot := domain.AvailableSlot{
		Start: time.Date(saturday.Year(), saturday.Month(), saturday.Day(), 9, 0, 0, 0, time.UTC),
		End:   time.Date(saturday.Year(), saturday.Month(), saturday.Day(), 9, 30, 0, 0, time.UTC),
	}

	slotRepo := &configuredAvailabilityRepo{slots: []domain.AvailableSlot{mondaySlot, saturdaySlot}}
	vehicleRepo := &stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}}
	desiredDate := mondaySlot.Start

	uc := usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{dealership: weekdayDealership()},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		vehicleRepo,
		&stubCache{service: oilChangeService(), hit: true},
		slotRepo,
		&stubAppointmentRepo{},
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)

	req := &usecase.AvailableSlotsInput{
		DealershipID: "d-001",
		Services:     []string{"oil_change"},
		VehicleID:    "v-001",
		DesiredDate:  desiredDate,
	}
	out, err := uc.AvailableSlots(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.AvailableSlots) != 1 {
		t.Errorf("want 1 slot (Saturday filtered), got %d", len(out.AvailableSlots))
	}
}

func TestAvailableSlots_WeekendRejected(t *testing.T) {
	uc := usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{dealership: weekdayDealership()},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		&stubVehicleRepo{vehicle: &domain.Vehicle{ID: "v-001"}},
		&stubCache{service: oilChangeService(), hit: true},
		&configuredAvailabilityRepo{},
		&stubAppointmentRepo{},
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)
	req := availableSlotsReq()
	req.DesiredDate = nextWeekend(8)
	_, err := uc.AvailableSlots(context.Background(), req)
	if !errors.Is(err, usecase.ErrSlotNotOnWorkingDay) {
		t.Errorf("want ErrSlotNotOnWorkingDay, got %v", err)
	}
}

// ─── ListAppointments stubs ───────────────────────────────────────────────────

// listableAppointmentRepo overrides the List* methods on stubAppointmentRepo
// so each query path can be configured independently.
type listableAppointmentRepo struct {
	stubAppointmentRepo
	byCustomer      []domain.Appointment
	byCustomerErr   error
	byDealership    []domain.Appointment
	byDealershipErr error
	byBoth          []domain.Appointment
	byBothErr       error
}

func (r *listableAppointmentRepo) ListAppointmentsByCustomer(_ context.Context, _ string) ([]domain.Appointment, error) {
	return r.byCustomer, r.byCustomerErr
}
func (r *listableAppointmentRepo) ListAppointmentsByDealership(_ context.Context, _ string) ([]domain.Appointment, error) {
	return r.byDealership, r.byDealershipErr
}
func (r *listableAppointmentRepo) ListAppointmentsByCustomerAndDealership(_ context.Context, _, _ string) ([]domain.Appointment, error) {
	return r.byBoth, r.byBothErr
}

func newListUC(apptRepo *listableAppointmentRepo) *usecase.AppoinmentBookingUseCase {
	return usecase.NewAppointmentBookingUseCase(
		&stubDealershipRepo{},
		&stubServiceBayRepo{},
		&stubTechnicianRepo{},
		&stubServiceDefRepo{},
		&stubVehicleRepo{},
		&stubCache{},
		&stubAvailabilityRepo{},
		apptRepo,
		defaultBookingRepo(),
		&stubBookingLocker{acquired: true},
		&stubTracer{},
	)
}

// sampleAppointment returns a fully-populated domain.Appointment for field-mapping tests.
func sampleAppointment(id, customerID, vehicleID, dealershipID string) domain.Appointment {
	start := futureTime(10)
	return domain.Appointment{
		ID:           id,
		CustomerID:   customerID,
		VehicleID:    vehicleID,
		DealershipID: dealershipID,
		Services: []*domain.ServiceSnapshot{
			{ServiceID: "svc-oil", Name: "Oil Change", EstimatedMinutes: 30, Price: 29.99},
		},
		Status:    domain.AppointmentStatusPending,
		StartTime: start,
		EndTime:   start.Add(30 * time.Minute),
		Notes:     "check brakes",
		CreatedAt: start.Add(-24 * time.Hour),
	}
}

// ─── ListAppointments tests ───────────────────────────────────────────────────

func TestListAppointments_ByCustomer_ReturnsAppointments(t *testing.T) {
	appts := []domain.Appointment{
		sampleAppointment("appt-1", "c-001", "v-001", "d-001"),
		sampleAppointment("appt-2", "c-001", "v-002", "d-002"),
	}
	uc := newListUC(&listableAppointmentRepo{byCustomer: appts})

	out, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{CustomerID: "c-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Appointments) != 2 {
		t.Errorf("want 2 appointments, got %d", len(out.Appointments))
	}
}

func TestListAppointments_ByDealership_ReturnsAppointments(t *testing.T) {
	appts := []domain.Appointment{
		sampleAppointment("appt-3", "c-001", "v-001", "d-001"),
	}
	uc := newListUC(&listableAppointmentRepo{byDealership: appts})

	out, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{DealershipID: "d-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Appointments) != 1 {
		t.Errorf("want 1 appointment, got %d", len(out.Appointments))
	}
}

func TestListAppointments_ByCustomerAndDealership_ReturnsAppointments(t *testing.T) {
	appts := []domain.Appointment{
		sampleAppointment("appt-4", "c-001", "v-001", "d-001"),
		sampleAppointment("appt-5", "c-001", "v-003", "d-001"),
	}
	uc := newListUC(&listableAppointmentRepo{byBoth: appts})

	out, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{
		CustomerID:   "c-001",
		DealershipID: "d-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Appointments) != 2 {
		t.Errorf("want 2 appointments, got %d", len(out.Appointments))
	}
}

func TestListAppointments_RepoError_ByCustomer(t *testing.T) {
	repoErr := errors.New("db failure")
	uc := newListUC(&listableAppointmentRepo{byCustomerErr: repoErr})

	_, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{CustomerID: "c-001"})
	if !errors.Is(err, repoErr) {
		t.Errorf("want repo error propagated, got %v", err)
	}
}

func TestListAppointments_RepoError_ByDealership(t *testing.T) {
	repoErr := errors.New("db failure")
	uc := newListUC(&listableAppointmentRepo{byDealershipErr: repoErr})

	_, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{DealershipID: "d-001"})
	if !errors.Is(err, repoErr) {
		t.Errorf("want repo error propagated, got %v", err)
	}
}

func TestListAppointments_RepoError_ByCustomerAndDealership(t *testing.T) {
	repoErr := errors.New("db failure")
	uc := newListUC(&listableAppointmentRepo{byBothErr: repoErr})

	_, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{
		CustomerID:   "c-001",
		DealershipID: "d-001",
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("want repo error propagated, got %v", err)
	}
}

func TestListAppointments_EmptyResult_ReturnsEmptySlice(t *testing.T) {
	uc := newListUC(&listableAppointmentRepo{byCustomer: []domain.Appointment{}})

	out, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{CustomerID: "c-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Appointments) != 0 {
		t.Errorf("want 0 appointments, got %d", len(out.Appointments))
	}
}

func TestListAppointments_FieldMapping(t *testing.T) {
	src := sampleAppointment("appt-99", "c-001", "v-001", "d-001")
	uc := newListUC(&listableAppointmentRepo{byCustomer: []domain.Appointment{src}})

	out, err := uc.ListAppointments(context.Background(), &usecase.ListAppointmentsInput{CustomerID: "c-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Appointments) != 1 {
		t.Fatalf("want 1 appointment, got %d", len(out.Appointments))
	}
	got := out.Appointments[0]

	if got.ID != src.ID {
		t.Errorf("ID: got %q, want %q", got.ID, src.ID)
	}
	if got.CustomerID != src.CustomerID {
		t.Errorf("CustomerID: got %q, want %q", got.CustomerID, src.CustomerID)
	}
	if got.VehicleID != src.VehicleID {
		t.Errorf("VehicleID: got %q, want %q", got.VehicleID, src.VehicleID)
	}
	if got.DealershipID != src.DealershipID {
		t.Errorf("DealershipID: got %q, want %q", got.DealershipID, src.DealershipID)
	}
	if got.Status != string(src.Status) {
		t.Errorf("Status: got %q, want %q", got.Status, string(src.Status))
	}
	if !got.StartTime.Equal(src.StartTime) {
		t.Errorf("StartTime: got %v, want %v", got.StartTime, src.StartTime)
	}
	if !got.EndTime.Equal(src.EndTime) {
		t.Errorf("EndTime: got %v, want %v", got.EndTime, src.EndTime)
	}
	if got.Notes != src.Notes {
		t.Errorf("Notes: got %q, want %q", got.Notes, src.Notes)
	}
	if !got.CreatedAt.Equal(src.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", got.CreatedAt, src.CreatedAt)
	}
	if len(got.Services) != len(src.Services) {
		t.Errorf("Services length: got %d, want %d", len(got.Services), len(src.Services))
	}
}

// multiServiceDefRepo supports per-ID service lookups for multi-service tests.
type multiServiceDefRepo struct {
	services map[string]*domain.Service
}

func (m *multiServiceDefRepo) GetService(_ context.Context, id string) (*domain.Service, error) {
	if svc, ok := m.services[id]; ok {
		return svc, nil
	}
	return nil, errors.New("not found")
}
func (m *multiServiceDefRepo) ListServices(_ context.Context) ([]*domain.Service, error) {
	return nil, nil
}
func (m *multiServiceDefRepo) CreateService(_ context.Context, _ *domain.Service) error { return nil }
func (m *multiServiceDefRepo) UpdateService(_ context.Context, _ *domain.Service) error { return nil }
func (m *multiServiceDefRepo) DeleteService(_ context.Context, _ string) error          { return nil }
