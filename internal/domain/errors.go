package domain

import "errors"

// Appointment errors
var (
	ErrAppointmentNotFound = errors.New("appointment not found")
	ErrTimeSlotConflict    = errors.New("requested time has been taken ,please choose another time")
	ErrInvalidTimeRange    = errors.New("end time must be after start time")
	ErrPastAppointmentTime = errors.New("cannot book appointment in the past")
)

// Resource availability errors
var (
	ErrNoAvailableBay        = errors.New("no available service bay for the requested time")
	ErrNoAvailableTechnician = errors.New("no qualified technician available for the requested time")
	ErrNoAvailableResources  = errors.New("slot taken, please choose another slot")
	// ErrDuplicateBooking is returned by the repository when a unique or exclusion
	// constraint fires on a plain INSERT, meaning a concurrent request won the race.
	ErrDuplicateBooking = errors.New("booking conflict: slot already taken by a concurrent request")
)

// Entity not found errors
var (
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrVehicleNotFound    = errors.New("vehicle not found")
	ErrDealershipNotFound = errors.New("dealership not found")
	ErrServiceNotFound    = errors.New("service definition not found")
	ErrTechnicianNotFound = errors.New("technician not found")
	ErrServiceBayNotFound = errors.New("service bay not found")
)

// Validation errors
var (
	ErrInvalidServiceType        = errors.New("invalid or unsupported service type")
	ErrVehicleNotOwnedByCustomer = errors.New("vehicle does not belong to the specified customer")
	ErrDealershipInactive        = errors.New("dealership is not currently active")
	ErrTechnicianNotQualified    = errors.New("technician is not qualified for the requested service type")
)
