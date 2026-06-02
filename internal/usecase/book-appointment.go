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

	if req.DesiredStartTime.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}

	services, totalDuration, err := uc.resolveServices(ctx, req.Services)
	if err != nil {
		return nil, err
	}

	rCtx, rSpan := uc.tracer.Start(ctx, "repo.GetDealership")
	dealership, repoErr := uc.dealershipRepo.GetDealership(rCtx, req.DealershipID)
	if repoErr != nil {
		rSpan.RecordError(repoErr)
		rSpan.SetStatus(codes.Error, repoErr.Error())
	}
	rSpan.End()
	if repoErr != nil {
		return nil, ErrDealershipNotFound
	}
	if !dealership.IsWorkingDay(req.DesiredStartTime) {
		return nil, ErrSlotNotOnWorkingDay
	}
	if !dealership.IsSlotWithinHours(req.DesiredStartTime, req.DesiredStartTime.Add(totalDuration)) {
		return nil, ErrSlotOutsideWorkingHours
	}

	rCtx, rSpan = uc.tracer.Start(ctx, "repo.GetVehicle")
	vehicle, repoErr := uc.vehicleRepo.GetVehicle(rCtx, req.VehicleID)
	if repoErr != nil {
		rSpan.RecordError(repoErr)
		rSpan.SetStatus(codes.Error, repoErr.Error())
	}
	rSpan.End()
	if repoErr != nil {
		return nil, ErrVehicleNotFound
	}
	if vehicle.CustomerID != req.CustomerID {
		return nil, ErrVehicleNotOwnedByCustomer
	}

	rCtx, rSpan = uc.tracer.Start(ctx, "repo.CreateAppointment")
	appt, err := uc.appointmentRepo.CreateAppointment(rCtx, &domain.Appointment{
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
		rSpan.RecordError(err)
		rSpan.SetStatus(codes.Error, err.Error())
	}
	rSpan.End()
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

	if req.DesiredDate.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}

	_, totalDuration, err := uc.resolveServices(ctx, req.Services)
	if err != nil {
		return nil, err
	}

	rCtx, rSpan := uc.tracer.Start(ctx, "repo.GetDealership")
	dealership, repoErr := uc.dealershipRepo.GetDealership(rCtx, req.DealershipID)
	if repoErr != nil {
		rSpan.RecordError(repoErr)
		rSpan.SetStatus(codes.Error, repoErr.Error())
	}
	rSpan.End()
	if repoErr != nil {
		return nil, ErrDealershipNotFound
	}
	if !dealership.IsWorkingDay(req.DesiredDate) {
		return nil, ErrSlotNotOnWorkingDay
	}

	rCtx, rSpan = uc.tracer.Start(ctx, "repo.GetVehicle")
	vehicle, repoErr := uc.vehicleRepo.GetVehicle(rCtx, req.VehicleID)
	if repoErr != nil {
		rSpan.RecordError(repoErr)
		rSpan.SetStatus(codes.Error, repoErr.Error())
	}
	rSpan.End()
	if repoErr != nil {
		return nil, ErrVehicleNotFound
	}
	if vehicle.CustomerID != req.CustomerID {
		return nil, ErrVehicleNotOwnedByCustomer
	}

	rCtx, rSpan = uc.tracer.Start(ctx, "repo.GetAvailableSlots")
	rawSlots, err := uc.availabilitySlotRepo.GetAvailableSlots(rCtx, req.DesiredDate, req.DesiredDate.Add(timeSlotLookupWindow), req.DealershipID, totalDuration, req.Services)
	if err != nil {
		rSpan.RecordError(err)
		rSpan.SetStatus(codes.Error, err.Error())
	}
	rSpan.End()
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
			rCtx, rSpan := uc.tracer.Start(ctx, "repo.GetService")
			svc, err = uc.serviceRepo.GetService(rCtx, id)
			if err != nil {
				rSpan.RecordError(err)
				rSpan.SetStatus(codes.Error, err.Error())
				rSpan.End()
				return nil, 0, fmt.Errorf("service %s not found", id)
			}
			rSpan.End()
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
		rCtx, rSpan := uc.tracer.Start(ctx, "repo.ListAppointmentsByCustomerAndDealership")
		appts, err = uc.appointmentRepo.ListAppointmentsByCustomerAndDealership(rCtx, req.CustomerID, req.DealershipID)
		if err != nil {
			rSpan.RecordError(err)
			rSpan.SetStatus(codes.Error, err.Error())
		}
		rSpan.End()
	case req.CustomerID != "":
		rCtx, rSpan := uc.tracer.Start(ctx, "repo.ListAppointmentsByCustomer")
		appts, err = uc.appointmentRepo.ListAppointmentsByCustomer(rCtx, req.CustomerID)
		if err != nil {
			rSpan.RecordError(err)
			rSpan.SetStatus(codes.Error, err.Error())
		}
		rSpan.End()
	case req.DealershipID != "":
		rCtx, rSpan := uc.tracer.Start(ctx, "repo.ListAppointmentsByDealership")
		appts, err = uc.appointmentRepo.ListAppointmentsByDealership(rCtx, req.DealershipID)
		if err != nil {
			rSpan.RecordError(err)
			rSpan.SetStatus(codes.Error, err.Error())
		}
		rSpan.End()
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
