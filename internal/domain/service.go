package domain

import "time"

// ServiceType represents the type of service to be performed.
type ServiceType string

const (
	ServiceOilChange       ServiceType = "oil_change"
	ServiceTireRotation    ServiceType = "tire_rotation"
	ServiceBrakeInspection ServiceType = "brake_inspection"
	ServiceEngineDiag      ServiceType = "engine_diagnostics"
	ServiceFullService     ServiceType = "full_service"
	ServiceMOT             ServiceType = "mot_inspection"
)

// Service describes a catalogue entry for a service that a
// dealership can offer. It includes the estimated duration used to
// calculate appointment end times.
type Service struct {
	ID                string
	Name              string
	Type              ServiceType
	Description       string
	EstimatedMinutes  int
	Price             float64
	IsActive          bool
	DealershipService bool
	DeletedAt         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EstimatedDuration returns the estimated duration as a time.Duration.
func (sd *Service) EstimatedDuration() time.Duration {
	return time.Duration(sd.EstimatedMinutes) * time.Minute
}
