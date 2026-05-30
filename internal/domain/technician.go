package domain

import "time"

// TechnicianStatus represents the employment/availability status of a technician.
type TechnicianStatus string

const (
	TechnicianStatusActive   TechnicianStatus = "active"
	TechnicianStatusInactive TechnicianStatus = "inactive"
	TechnicianStatusOnLeave  TechnicianStatus = "on_leave"
)

// Technician represents a qualified service technician at a dealership.
// A technician has a set of skills (service types they are qualified to perform)
// and can only work on one appointment at a time.
type Technician struct {
	ID           string
	DealershipID string
	FirstName    string
	LastName     string
	Skills       []ServiceType
	Status       TechnicianStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FullName returns the technician's full display name.
func (t *Technician) FullName() string {
	return t.FirstName + " " + t.LastName
}

// IsQualifiedFor checks if the technician has the skill to perform the given service type.
func (t *Technician) IsQualifiedFor(serviceType ServiceType) bool {
	for _, skill := range t.Skills {
		if skill == serviceType {
			return true
		}
	}
	return false
}

// IsActive returns true if the technician is currently working and available.
func (t *Technician) IsActive() bool {
	return t.Status == TechnicianStatusActive
}
