package usecase

import (
	"context"
	"errors"
	"fmt"
	"keyloop-test/internal/domain"
	"keyloop-test/internal/repository"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const serviceCacheTTL = 5 * time.Minute

var (
	ErrDealershipNotFound        = errors.New("dealership not found")
	ErrVehicleNotFound           = errors.New("vehicle not found")
	ErrVehicleNotOwnedByCustomer = errors.New("vehicle is not owned by the specified customer")
	ErrDesiredDateInPast         = errors.New("desired date must be in the future")
	ErrStartTimeInPast           = errors.New("appointment start time must be in the future")
	ErrDealershipIDRequired      = errors.New("dealership_id is required")
	ErrServiceTypesRequired      = errors.New("at least one service type is required")
	ErrListFilterRequired        = errors.New("customer_id or dealership_id is required")
	ErrSlotOutsideWorkingHours   = errors.New("chosen slot is outside dealership working hours")
	ErrSlotNotOnWorkingDay       = errors.New("chosen date is not a working day for this dealership")
)

type AppoinmentBookingUseCase struct {
	dealershipRepo       repository.DealershipRepository
	serviceBayRepo       repository.ServiceBayRepository
	technicianRepo       repository.TechnicianRepository
	serviceRepo          repository.ServiceRepository
	vehicleRepo          repository.VehicleRepository
	serviceCache         ServiceCacheProvider
	availabilitySlotRepo repository.AvailabilitySlotRepository
	appointmentRepo      repository.AppointmentRepository
	tracer               trace.Tracer
}

// NewAppointmentBookingUseCase creates a new instance of AppoinmentBookingUseCase.
func NewAppointmentBookingUseCase(
	dealershipRepo repository.DealershipRepository,
	serviceBayRepo repository.ServiceBayRepository,
	technicianRepo repository.TechnicianRepository,
	serviceRepo repository.ServiceRepository,
	vehicleRepo repository.VehicleRepository,
	serviceCache ServiceCacheProvider,
	availabilitySlotRepo repository.AvailabilitySlotRepository,
	appointmentRepo repository.AppointmentRepository,
) *AppoinmentBookingUseCase {
	return &AppoinmentBookingUseCase{
		dealershipRepo:       dealershipRepo,
		serviceBayRepo:       serviceBayRepo,
		technicianRepo:       technicianRepo,
		serviceRepo:          serviceRepo,
		vehicleRepo:          vehicleRepo,
		serviceCache:         serviceCache,
		availabilitySlotRepo: availabilitySlotRepo,
		appointmentRepo:      appointmentRepo,
		tracer:               otel.Tracer("keyloop-test/usecase"),
	}
}

type AppoinmentBookingInput struct {
	CustomerID       string
	DealershipID     string
	Services         []string
	DesiredStartTime time.Time
	VehicleID        string
	Notes            string
}

type AppoinmentBookingOutput struct {
	AppointmentID   string
	DurationMinutes int
	Message         string
}

// WarmCache loads all services from the DB into the cache.
// Call once at startup so the first requests don't all hit the database.
func (uc *AppoinmentBookingUseCase) WarmCache(ctx context.Context) error {
	services, err := uc.serviceRepo.ListServices(ctx)
	if err != nil {
		return err
	}
	for _, svc := range services {
		if err := uc.serviceCache.SetService(ctx, svc.ID, svc, serviceCacheTTL); err != nil {
			return err
		}
	}
	return nil
}

func (uc *AppoinmentBookingUseCase) BookAppointment(ctx context.Context, req *AppoinmentBookingInput) (_ *AppoinmentBookingOutput, err error) {
	ctx, span := uc.tracer.Start(ctx, "BookAppointment",
		trace.WithAttributes(
			attribute.String("dealership.id", req.DealershipID),
			attribute.String("vehicle.id", req.VehicleID),
			attribute.Int("services.count", len(req.Services)),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	//Validate input
	if req.DealershipID == "" {
		return nil, ErrDealershipIDRequired
	}
	if len(req.Services) == 0 {
		return nil, ErrServiceTypesRequired
	}
	if req.DesiredStartTime.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}

	//Get services snapshots and total duration for all requested services
	services, totalDuration, err := uc.resolveServices(ctx, req.Services)
	if err != nil {
		return nil, err
	}

	dealership, err := uc.dealershipRepo.GetDealership(ctx, req.DealershipID)
	if err != nil {
		return nil, ErrDealershipNotFound
	}
	if !dealership.IsWorkingDay(req.DesiredStartTime) {
		return nil, ErrSlotNotOnWorkingDay
	}
	if !dealership.IsSlotWithinHours(req.DesiredStartTime, req.DesiredStartTime.Add(totalDuration)) {
		return nil, ErrSlotOutsideWorkingHours
	}

	//Check vehicle exists and belongs to customer
	vehicle, err := uc.vehicleRepo.GetVehicle(ctx, req.VehicleID)
	if err != nil {
		return nil, ErrVehicleNotFound
	}
	if vehicle.CustomerID != req.CustomerID {
		return nil, ErrVehicleNotOwnedByCustomer
	}

	appt, err := uc.appointmentRepo.CreateAppointment(ctx, &domain.Appointment{
		CustomerID:   req.CustomerID,
		VehicleID:    req.VehicleID,
		DealershipID: req.DealershipID,
		Status:       domain.AppointmentStatusPending,
		StartTime:    req.DesiredStartTime,
		EndTime:      req.DesiredStartTime.Add(totalDuration),
		Services:     services,
		Notes:        req.Notes,
	})
	if err != nil {
		return nil, err
	}

	return &AppoinmentBookingOutput{
		AppointmentID:   appt.ID,
		DurationMinutes: int(appt.Duration().Minutes()),
		Message:         "Appointment booked successfully",
	}, nil
}

type AvailableSlotsInput struct {
	CustomerID   string
	DealershipID string
	Services     []string
	VehicleID    string
	DesiredDate  time.Time
}

type AvailableSlotsOutput struct {
	AvailableSlots []AvailableSlot
}

type AvailableSlot struct {
	Start time.Time
	End   time.Time
}

const timeSlotLookupWindow = 7 * 24 * time.Hour

func (uc *AppoinmentBookingUseCase) AvailableSlots(ctx context.Context, req *AvailableSlotsInput) (_ *AvailableSlotsOutput, err error) {
	ctx, span := uc.tracer.Start(ctx, "AvailableSlots",
		trace.WithAttributes(
			attribute.String("dealership.id", req.DealershipID),
			attribute.String("vehicle.id", req.VehicleID),
			attribute.Int("services.count", len(req.Services)),
			attribute.String("desired_date", req.DesiredDate.Format(time.DateOnly)),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	//Validate input
	if req.DealershipID == "" {
		return nil, ErrDealershipIDRequired
	}
	if len(req.Services) == 0 {
		return nil, ErrServiceTypesRequired
	}
	if req.DesiredDate.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}

	//Get total duration for all requested services
	_, totalDuration, err := uc.resolveServices(ctx, req.Services)
	if err != nil {
		return nil, err
	}

	//Check dealership exists and is open on desired date
	dealership, err := uc.dealershipRepo.GetDealership(ctx, req.DealershipID)
	if err != nil {
		return nil, ErrDealershipNotFound
	}
	if !dealership.IsWorkingDay(req.DesiredDate) {
		return nil, ErrSlotNotOnWorkingDay
	}

	//Check vehicle exists and belongs to customer
	vehicle, err := uc.vehicleRepo.GetVehicle(ctx, req.VehicleID)
	if err != nil {
		return nil, ErrVehicleNotFound
	}
	if vehicle.CustomerID != req.CustomerID {
		return nil, ErrVehicleNotOwnedByCustomer
	}

	rawSlots, err := uc.availabilitySlotRepo.GetAvailableSlots(ctx, req.DesiredDate, req.DesiredDate.Add(timeSlotLookupWindow), req.DealershipID, totalDuration, req.Services)
	if err != nil {
		return nil, err
	}

	filtered := make([]AvailableSlot, 0, len(rawSlots))
	for _, s := range rawSlots {
		if dealership.IsSlotWithinHours(s.Start, s.End) && dealership.IsWorkingDay(s.Start) {
			filtered = append(filtered, AvailableSlot{Start: s.Start, End: s.End})
		}
	}

	return &AvailableSlotsOutput{AvailableSlots: filtered}, nil
}

// resolveServices fetches each service from cache (falling back to DB on miss)
// and returns snapshots plus the combined estimated duration.
func (uc *AppoinmentBookingUseCase) resolveServices(ctx context.Context, serviceIDs []string) ([]*domain.ServiceSnapshot, time.Duration, error) {
	snapshots := make([]*domain.ServiceSnapshot, 0, len(serviceIDs))
	var total time.Duration
	for _, id := range serviceIDs {
		svc, ok, err := uc.serviceCache.GetService(ctx, id)
		if !ok || err != nil {
			svc, err = uc.serviceRepo.GetService(ctx, id)
			if err != nil {
				return nil, 0, fmt.Errorf("service %s not found", id)
			}
			uc.serviceCache.SetService(ctx, id, svc, serviceCacheTTL)
		}
		snapshots = append(snapshots, &domain.ServiceSnapshot{
			ServiceID:        svc.ID,
			Name:             svc.Name,
			EstimatedMinutes: svc.EstimatedMinutes,
			Price:            svc.Price,
		})
		total += time.Duration(svc.EstimatedMinutes) * time.Minute
	}
	return snapshots, total, nil
}

type ListAppointmentsInput struct {
	CustomerID   string
	DealershipID string
}

type AppointmentItem struct {
	ID           string
	CustomerID   string
	VehicleID    string
	DealershipID string
	Services     []*domain.ServiceSnapshot
	Status       string
	StartTime    time.Time
	EndTime      time.Time
	Notes        string
	CreatedAt    time.Time
}

type ListAppointmentsOutput struct {
	Appointments []AppointmentItem
}

func (uc *AppoinmentBookingUseCase) ListAppointments(ctx context.Context, req *ListAppointmentsInput) (_ *ListAppointmentsOutput, err error) {
	ctx, span := uc.tracer.Start(ctx, "ListAppointments",
		trace.WithAttributes(
			attribute.String("customer.id", req.CustomerID),
			attribute.String("dealership.id", req.DealershipID),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	var appts []domain.Appointment
	switch {
	case req.CustomerID != "" && req.DealershipID != "":
		appts, err = uc.appointmentRepo.ListAppointmentsByCustomerAndDealership(ctx, req.CustomerID, req.DealershipID)
	case req.CustomerID != "":
		appts, err = uc.appointmentRepo.ListAppointmentsByCustomer(ctx, req.CustomerID)
	case req.DealershipID != "":
		appts, err = uc.appointmentRepo.ListAppointmentsByDealership(ctx, req.DealershipID)
	default:
		return nil, ErrListFilterRequired
	}
	if err != nil {
		return nil, err
	}
	items := make([]AppointmentItem, len(appts))
	for i, a := range appts {
		items[i] = AppointmentItem{
			ID:           a.ID,
			CustomerID:   a.CustomerID,
			VehicleID:    a.VehicleID,
			DealershipID: a.DealershipID,
			Services:     a.Services,
			Status:       string(a.Status),
			StartTime:    a.StartTime,
			EndTime:      a.EndTime,
			Notes:        a.Notes,
			CreatedAt:    a.CreatedAt,
		}
	}
	return &ListAppointmentsOutput{Appointments: items}, nil
}
