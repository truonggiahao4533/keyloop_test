package domain

import "time"

// AppointmentStatus represents the lifecycle status of an appointment.
type AppointmentStatus string

const (
	AppointmentStatusPending   AppointmentStatus = "pending"
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
	AppointmentStatusCompleted AppointmentStatus = "completed"
	AppointmentStatusNoShow    AppointmentStatus = "no_show"
)

// Appointment is the core aggregate root — a confirmed booking that
// associates a Customer, Vehicle, Technician, and ServiceBay at a
// specific Dealership for a given service and time window.
type Appointment struct {
	ID           string
	CustomerID   string
	VehicleID    string
	DealershipID string
	ServiceBayID string
	TechnicianID string
	Services     []ServiceSnapshot
	Status       AppointmentStatus
	StartTime    time.Time
	EndTime      time.Time
	Notes        string
	DeletedAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ServiceSnapshot struct {
	ServiceID        string
	Name             string
	EstimatedMinutes int
	Price            float64
}

// Duration returns the total duration of the appointment.
func (a *Appointment) Duration() time.Duration {
	return a.EndTime.Sub(a.StartTime)
}

// IsConfirmed returns true if the appointment is in a confirmed state.
func (a *Appointment) IsConfirmed() bool {
	return a.Status == AppointmentStatusConfirmed
}

// CanCancel returns true if the appointment is in a state that allows cancellation.
func (a *Appointment) CanCancel() bool {
	return a.Status == AppointmentStatusPending || a.Status == AppointmentStatusConfirmed
}

// OverlapsWith checks if this appointment's time window overlaps with the given range.
func (a *Appointment) OverlapsWith(start, end time.Time) bool {
	return a.StartTime.Before(end) && a.EndTime.After(start)
}
