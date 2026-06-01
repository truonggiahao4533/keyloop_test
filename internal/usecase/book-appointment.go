package usecase

import (
	"context"
	"errors"
	"keyloop-test/internal/domain"
	"keyloop-test/internal/repository"
	"time"
)

const serviceCacheTTL = 5 * time.Minute

var (
	ErrDealershipNotFound   = errors.New("dealership not found")
	ErrVehicleNotFound      = errors.New("vehicle not found")
	ErrDesiredDateInPast    = errors.New("desired date must be in the future")
	ErrStartTimeInPast      = errors.New("appointment start time must be in the future")
	ErrDealershipIDRequired = errors.New("dealership_id is required")
	ErrServiceTypesRequired = errors.New("at least one service type is required")
)

type AppoinmentBookingUseCase struct {
	dealershipRepo        repository.DealershipRepository
	serviceBayRepo        repository.ServiceBayRepository
	technicianRepo        repository.TechnicianRepository
	serviceDefinitionRepo repository.ServiceDefinitionRepository
	vehicleRepo           repository.VehicleRepository
	serviceCache          ServiceCacheProvider
	availabilitySlotRepo  repository.AvailabilitySlotRepository
	appointmentRepo       repository.AppointmentRepository
}

// NewAppointmentBookingUseCase creates a new instance of AppoinmentBookingUseCase.
func NewAppointmentBookingUseCase(
	dealershipRepo repository.DealershipRepository,
	serviceBayRepo repository.ServiceBayRepository,
	technicianRepo repository.TechnicianRepository,
	serviceDefinitionRepo repository.ServiceDefinitionRepository,
	vehicleRepo repository.VehicleRepository,
	serviceCache ServiceCacheProvider,
	availabilitySlotRepo repository.AvailabilitySlotRepository,
	appointmentRepo repository.AppointmentRepository,
) *AppoinmentBookingUseCase {
	return &AppoinmentBookingUseCase{
		dealershipRepo:        dealershipRepo,
		serviceBayRepo:        serviceBayRepo,
		technicianRepo:        technicianRepo,
		serviceDefinitionRepo: serviceDefinitionRepo,
		vehicleRepo:           vehicleRepo,
		serviceCache:          serviceCache,
		availabilitySlotRepo:  availabilitySlotRepo,
		appointmentRepo:       appointmentRepo,
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

func (uc *AppoinmentBookingUseCase) BookAppointment(ctx context.Context, req *AppoinmentBookingInput) (*AppoinmentBookingOutput, error) {

	//Check if dealership is valid
	if req.DealershipID == "" {
		return nil, ErrDealershipIDRequired
	}
	//Check if service type is valid (Assume all dealerships offer the same services for simplicity)
	if len(req.Services) == 0 {
		return nil, ErrServiceTypesRequired
	}

	desiredStartTime := req.DesiredStartTime
	var services []*domain.ServiceSnapshot
	var totalDuration time.Duration
	for _, s := range req.Services {
		service, ok, err := uc.serviceCache.GetService(ctx, s)
		//If cache miss or error, get from DB and populate cache. If DB error, return error
		if !ok || err != nil {
			//Cache miss, get from DB and populate cache
			def, err := uc.serviceDefinitionRepo.GetServiceDefinition(ctx, s)
			if err != nil {
				return nil, errors.New("service " + s + " not found")
			}
			service = &domain.Service{
				ID:               def.ID,
				Name:             def.Name,
				EstimatedMinutes: def.EstimatedMinutes,
				Price:            def.Price,
			}
			uc.serviceCache.SetService(ctx, s, service, serviceCacheTTL)
		}
		services = append(services, &domain.ServiceSnapshot{
			ServiceID:        service.ID,
			Name:             service.Name,
			EstimatedMinutes: service.EstimatedMinutes,
			Price:            service.Price,
		})
		totalDuration += time.Duration(service.EstimatedMinutes) * time.Minute
	}

	//Check if desired date is in the past
	if req.DesiredStartTime.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}
	//Check if chosen service's duration matches dealership working hours and desired date
	dealership, err := uc.dealershipRepo.GetDealership(ctx, req.DealershipID)
	if err != nil {
		return nil, ErrDealershipNotFound
	}
	if !dealership.IsSlotWithinHours(req.DesiredStartTime, req.DesiredStartTime.Add(totalDuration)) {
		return nil, errors.New("chosen slot is outside dealership working hours")
	}

	//Check if vehicle is valid
	_, err = uc.vehicleRepo.GetVehicle(ctx, req.VehicleID)
	if err != nil {
		return nil, ErrVehicleNotFound
	}

	//Create appointment
	//Start with pending status, then after all checks are done, update to confirmed
	appt, err := uc.appointmentRepo.CreateAppointment(ctx, &domain.Appointment{
		CustomerID:   req.CustomerID,
		VehicleID:    req.VehicleID,
		DealershipID: req.DealershipID,
		Status:       domain.AppointmentStatusPending,
		EndTime:      desiredStartTime.Add(totalDuration),
		StartTime:    req.DesiredStartTime,
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
	CustomerID    string
	DealershipID  string
	Services      []string
	TotalDuration time.Duration
	VehicleID     string
	DesiredDate   time.Time
}

type AvailableSlotsOutput struct {
	AvailableSlots []AvailableSlot
}

type AvailableSlot struct {
	Start time.Time
	End   time.Time
}

const timeSlotLookupWindow = 7 * time.Hour * 24 // 7 days

func (uc *AppoinmentBookingUseCase) AvailableSlots(ctx context.Context, req *AvailableSlotsInput) (*AvailableSlotsOutput, error) {
	//TODO: Check if dealership is valid
	if req.DealershipID == "" {
		return nil, ErrDealershipIDRequired
	}
	if len(req.Services) == 0 {
		return nil, ErrServiceTypesRequired
	}
	if req.DesiredDate.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}

	dealership, err := uc.dealershipRepo.GetDealership(ctx, req.DealershipID)
	if err != nil {
		return nil, ErrDealershipNotFound
	}

	_, err = uc.vehicleRepo.GetVehicle(ctx, req.VehicleID)
	if err != nil {
		return nil, ErrVehicleNotFound
	}

	startTime := req.DesiredDate
	//Assume available slot is within 7 days
	endTime := req.DesiredDate.Add(7 * 24 * time.Hour)

	rawSlots, err := uc.availabilitySlotRepo.GetAvailableSlots(ctx, startTime, endTime, req.DealershipID, req.TotalDuration, req.Services)
	if err != nil {
		return nil, err
	}

	filtered := make([]AvailableSlot, 0, len(rawSlots))
	for _, s := range rawSlots {
		if dealership.IsSlotWithinHours(s.Start, s.End) {
			filtered = append(filtered, AvailableSlot{Start: s.Start, End: s.End})
		}
	}

	return &AvailableSlotsOutput{AvailableSlots: filtered}, nil
}
