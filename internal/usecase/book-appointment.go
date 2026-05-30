package usecase

import (
	"context"
	"errors"
	"keyloop-test/internal/repository"
	"time"
)

type AppoinmentBookingUseCase struct {
	dealershipRepo  repository.DealershipRepository
	serviceBayRepo  repository.ServiceBayRepository
	technicianRepo  repository.TechnicianRepository
	reservationRepo repository.ReservationRepository
}

// NewAppointmentBookingUseCase creates a new instance of AppoinmentBookingUseCase.
func NewAppointmentBookingUseCase(
	dealershipRepo repository.DealershipRepository,
	serviceBayRepo repository.ServiceBayRepository,
	technicianRepo repository.TechnicianRepository,
	reservationRepo repository.ReservationRepository,
) *AppoinmentBookingUseCase {
	return &AppoinmentBookingUseCase{
		dealershipRepo:  dealershipRepo,
		serviceBayRepo:  serviceBayRepo,
		technicianRepo:  technicianRepo,
		reservationRepo: reservationRepo,
	}
}

type AppoinmentBookingInput struct {
	DealershipID string
	ServiceType  []string
	StartTime    time.Time
	ServiceBayID string
	TechnicianID string
}

type AppoinmentBookingOutput struct {
	ReservationID string
}

func (uc *AppoinmentBookingUseCase) BookAppointment(ctx context.Context, req AppoinmentBookingInput) (AppoinmentBookingOutput, error) {
	//TODO: Validate inputs
	if req.StartTime.Before(time.Now()) {
		return AppoinmentBookingOutput{}, errors.New("appointment start time must be in the future")
	}
	_, err := uc.dealershipRepo.GetDealership(ctx, req.DealershipID)
	if err != nil {
		return AppoinmentBookingOutput{}, errors.New("dealership not found")
	}

	//TODO: Check if dealership is valid (isOpen ?)

	//TODO: Check if vehicle is valid (Check for existing vehicle)

	//TODO: Check if service type is valid (Check if the dealership offer this service)

	//TODO: Check if appointment is valid (Check for the requested time slot )

	//TODO: Check if appointment overlaps with existing appointments

	//TODO: Create appointment

	return AppoinmentBookingOutput{}, nil
}

type AvailableSlotsInput struct {
	DealershipID string
	ServiceType  []string
	StartTime    time.Time
	EndTime      time.Time
}

type AvailableSlotsOutput struct {
	AvailableSlots map[string]interface{}
}

func (uc *AppoinmentBookingUseCase) AvailableSlots(ctx context.Context, req AvailableSlotsInput) (AvailableSlotsOutput, error) {
	return AvailableSlotsOutput{}, nil
}
