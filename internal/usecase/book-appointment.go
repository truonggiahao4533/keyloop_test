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
	DealershipID string
	Services     []string
	DesiredDate  time.Time
	ServiceBayID string
	TechnicianID string
	ChosenSlot   ChosenSlot
	VehicleID    string
	Notes        string
}

type ChosenSlot struct {
	Start time.Time
	End   time.Time
}

type AppoinmentBookingOutput struct {
	AppointmentID string
	Message       string
}

func (uc *AppoinmentBookingUseCase) BookAppointment(ctx context.Context, req *AppoinmentBookingInput) (*AppoinmentBookingOutput, error) {
	//TODO: Validate inputs
	//TODO: Check if dealership is valid
	if req.DealershipID == "" {
		return nil, ErrDealershipIDRequired
	}
	//TODO: Check if service type is valid (Assume all dealerships offer the same services for simplicity)
	if len(req.Services) == 0 {
		return nil, ErrServiceTypesRequired
	}
	if req.DesiredDate.Before(time.Now()) {
		return nil, ErrDesiredDateInPast
	}

	//TODO: Check if vehicle is valid and belongs to the custmer
	_, err := uc.vehicleRepo.GetVehicle(ctx, req.VehicleID)
	if err != nil {
		return nil, ErrVehicleNotFound
	}

	//TODO: Create appointment
	//Start with pending status, then after all checks are done, update to confirmed
	appt, err := uc.appointmentRepo.CreateAppointment(ctx, &domain.Appointment{
		CustomerID:   "", //TODO: Get from auth context
		VehicleID:    req.VehicleID,
		DealershipID: req.DealershipID,
		ServiceBayID: req.ServiceBayID,
		TechnicianID: req.TechnicianID,
		Status:       domain.AppointmentStatusPending,
		StartTime:    req.ChosenSlot.Start,
		EndTime:      req.ChosenSlot.End,
		Notes:        req.Notes,
	})
	if err != nil {
		return nil, err
	}

	return &AppoinmentBookingOutput{AppointmentID: appt.ID, Message: "Appointment booked successfully"}, nil
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
