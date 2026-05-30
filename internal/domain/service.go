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

// ServiceDefinition describes a catalogue entry for a service that a
// dealership can offer. It includes the estimated duration used to
// calculate appointment end times.
type ServiceDefinition struct {
	ID               string
	Name             string
	Type             ServiceType
	Description      string
	EstimatedMinutes int
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// EstimatedDuration returns the estimated duration as a time.Duration.
func (sd *ServiceDefinition) EstimatedDuration() time.Duration {
	return time.Duration(sd.EstimatedMinutes) * time.Minute
}
