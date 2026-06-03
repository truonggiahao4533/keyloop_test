package usecase

import (
	"context"
	"errors"
	"keyloop-test/internal/domain"
	"keyloop-test/internal/port"
	"keyloop-test/internal/repository"
	"log/slog"
	"time"
)

const serviceCacheTTL = 5 * time.Minute

var (
	ErrDealershipNotFound        = errors.New("dealership not found")
	ErrVehicleNotFound           = errors.New("vehicle not found")
	ErrAppointmentNotFound       = errors.New("appointment not found")
	ErrVehicleNotOwnedByCustomer = errors.New("vehicle is not owned by the specified customer")
	ErrDesiredDateInPast         = errors.New("desired date must be in the future")
	ErrStartTimeInPast           = errors.New("appointment start time must be in the future")
	ErrSlotOutsideWorkingHours   = errors.New("chosen slot is outside dealership working hours")
	ErrSlotNotOnWorkingDay       = errors.New("chosen date is not a working day for this dealership")
	ErrInvalidStatusTransition   = errors.New("invalid status transition")
	ErrCannotDeleteAppointment   = errors.New("appointment cannot be deleted in its current status")
)

// validTransitions defines the allowed status transitions for an appointment.
var validTransitions = map[domain.AppointmentStatus]map[domain.AppointmentStatus]struct{}{
	domain.AppointmentStatusPending: {
		domain.AppointmentStatusConfirmed: {},
		domain.AppointmentStatusCancelled: {},
	},
	domain.AppointmentStatusConfirmed: {
		domain.AppointmentStatusCancelled: {},
		domain.AppointmentStatusCompleted: {},
		domain.AppointmentStatusNoShow:    {},
	},
}

type AppoinmentBookingUseCase struct {
	dealershipRepo       repository.DealershipRepository
	serviceBayRepo       repository.ServiceBayRepository
	technicianRepo       repository.TechnicianRepository
	serviceRepo          repository.ServiceRepository
	vehicleRepo          repository.VehicleRepository
	serviceCache         port.ServiceCacheProvider
	availabilitySlotRepo repository.AvailabilitySlotRepository
	appointmentRepo      repository.AppointmentRepository
	tracer               port.Tracer
}

// NewAppointmentBookingUseCase creates a new instance of AppoinmentBookingUseCase.
func NewAppointmentBookingUseCase(
	dealershipRepo repository.DealershipRepository,
	serviceBayRepo repository.ServiceBayRepository,
	technicianRepo repository.TechnicianRepository,
	serviceRepo repository.ServiceRepository,
	vehicleRepo repository.VehicleRepository,
	serviceCache port.ServiceCacheProvider,
	availabilitySlotRepo repository.AvailabilitySlotRepository,
	appointmentRepo repository.AppointmentRepository,
	tracer port.Tracer,
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
		tracer:               tracer,
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
		port.StringAttr("dealership.id", req.DealershipID),
		port.StringAttr("vehicle.id", req.VehicleID),
		port.IntAttr("services.count", len(req.Services)),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetErrorStatus(err.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "booking appointment",
		"dealership_id", req.DealershipID,
		"vehicle_id", req.VehicleID,
		"desired_start_time", req.DesiredStartTime,
		"service_count", len(req.Services),
	)

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
		rSpan.SetErrorStatus(repoErr.Error())
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
		rSpan.SetErrorStatus(repoErr.Error())
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
		rSpan.SetErrorStatus(err.Error())
	}
	rSpan.End()
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "appointment created",
		"appointment_id", appt.ID,
		"start_time", appt.StartTime,
		"end_time", appt.EndTime,
		"duration_minutes", int(appt.Duration().Minutes()),
	)
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
		port.StringAttr("dealership.id", req.DealershipID),
		port.StringAttr("vehicle.id", req.VehicleID),
		port.IntAttr("services.count", len(req.Services)),
		port.StringAttr("desired_date", req.DesiredDate.Format(time.DateOnly)),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetErrorStatus(err.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "finding available slots",
		"dealership_id", req.DealershipID,
		"desired_date", req.DesiredDate.Format(time.DateOnly),
		"service_count", len(req.Services),
	)

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
		rSpan.SetErrorStatus(repoErr.Error())
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
		rSpan.SetErrorStatus(repoErr.Error())
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
		rSpan.SetErrorStatus(err.Error())
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

	slog.InfoContext(ctx, "available slots computed", "count", len(filtered))
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
				rSpan.SetErrorStatus(err.Error())
				rSpan.End()
				return nil, 0, domain.ErrServiceNotFound
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
		port.StringAttr("customer.id", req.CustomerID),
		port.StringAttr("dealership.id", req.DealershipID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetErrorStatus(err.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "listing appointments",
		"customer_id", req.CustomerID,
		"dealership_id", req.DealershipID,
	)
	var appts []domain.Appointment
	switch {
	case req.CustomerID != "" && req.DealershipID != "":
		rCtx, rSpan := uc.tracer.Start(ctx, "repo.ListAppointmentsByCustomerAndDealership")
		appts, err = uc.appointmentRepo.ListAppointmentsByCustomerAndDealership(rCtx, req.CustomerID, req.DealershipID)
		if err != nil {
			rSpan.RecordError(err)
			rSpan.SetErrorStatus(err.Error())
		}
		rSpan.End()
	case req.CustomerID != "":
		rCtx, rSpan := uc.tracer.Start(ctx, "repo.ListAppointmentsByCustomer")
		appts, err = uc.appointmentRepo.ListAppointmentsByCustomer(rCtx, req.CustomerID)
		if err != nil {
			rSpan.RecordError(err)
			rSpan.SetErrorStatus(err.Error())
		}
		rSpan.End()
	case req.DealershipID != "":
		rCtx, rSpan := uc.tracer.Start(ctx, "repo.ListAppointmentsByDealership")
		appts, err = uc.appointmentRepo.ListAppointmentsByDealership(rCtx, req.DealershipID)
		if err != nil {
			rSpan.RecordError(err)
			rSpan.SetErrorStatus(err.Error())
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
	slog.InfoContext(ctx, "appointments listed", "count", len(items))
	return &ListAppointmentsOutput{Appointments: items}, nil
}

func (uc *AppoinmentBookingUseCase) SoftDeleteAppointment(ctx context.Context, id string) (err error) {
	ctx, span := uc.tracer.Start(ctx, "SoftDeleteAppointment",
		port.StringAttr("appointment.id", id),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetErrorStatus(err.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "deleting appointment", "appointment_id", id)
	rCtx, rSpan := uc.tracer.Start(ctx, "repo.GetAppointment")
	appt, repoErr := uc.appointmentRepo.GetAppointment(rCtx, id)
	if repoErr != nil {
		rSpan.RecordError(repoErr)
		rSpan.SetErrorStatus(repoErr.Error())
	}
	rSpan.End()
	if repoErr != nil {
		return ErrAppointmentNotFound
	}
	if !appt.CanCancel() {
		return ErrCannotDeleteAppointment
	}

	rCtx, rSpan = uc.tracer.Start(ctx, "repo.DeleteAppointment")
	err = uc.appointmentRepo.DeleteAppointment(rCtx, id)
	if err != nil {
		rSpan.RecordError(err)
		rSpan.SetErrorStatus(err.Error())
	}
	rSpan.End()
	if err == nil {
		slog.InfoContext(ctx, "appointment deleted", "appointment_id", id)
	}
	return err
}

type UpdateAppointmentInput struct {
	AppointmentID string
	Status        *domain.AppointmentStatus
	Notes         *string
}

func (uc *AppoinmentBookingUseCase) UpdateAppointment(ctx context.Context, req *UpdateAppointmentInput) (_ *AppointmentItem, err error) {
	ctx, span := uc.tracer.Start(ctx, "UpdateAppointment",
		port.StringAttr("appointment.id", req.AppointmentID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetErrorStatus(err.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "updating appointment", "appointment_id", req.AppointmentID)
	rCtx, rSpan := uc.tracer.Start(ctx, "repo.GetAppointment")
	current, repoErr := uc.appointmentRepo.GetAppointment(rCtx, req.AppointmentID)
	if repoErr != nil {
		rSpan.RecordError(repoErr)
		rSpan.SetErrorStatus(repoErr.Error())
	}
	rSpan.End()
	if repoErr != nil {
		return nil, ErrAppointmentNotFound
	}

	newStatus := current.Status
	if req.Status != nil && *req.Status != current.Status {
		allowed, ok := validTransitions[current.Status]
		if !ok {
			return nil, ErrInvalidStatusTransition
		}
		if _, ok := allowed[*req.Status]; !ok {
			return nil, ErrInvalidStatusTransition
		}
		slog.InfoContext(ctx, "appointment status transition",
			"appointment_id", req.AppointmentID,
			"from", string(current.Status),
			"to", string(*req.Status),
		)
		newStatus = *req.Status
	}

	newNotes := current.Notes
	if req.Notes != nil {
		newNotes = *req.Notes
	}

	rCtx, rSpan = uc.tracer.Start(ctx, "repo.UpdateAppointment")
	updated, err := uc.appointmentRepo.UpdateAppointment(rCtx, req.AppointmentID, newStatus, newNotes)
	if err != nil {
		rSpan.RecordError(err)
		rSpan.SetErrorStatus(err.Error())
	}
	rSpan.End()
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "appointment updated",
		"appointment_id", updated.ID,
		"status", string(updated.Status),
	)
	return &AppointmentItem{
		ID:           updated.ID,
		CustomerID:   updated.CustomerID,
		VehicleID:    updated.VehicleID,
		DealershipID: updated.DealershipID,
		Services:     updated.Services,
		Status:       string(updated.Status),
		StartTime:    updated.StartTime,
		EndTime:      updated.EndTime,
		Notes:        updated.Notes,
		CreatedAt:    updated.CreatedAt,
	}, nil
}
