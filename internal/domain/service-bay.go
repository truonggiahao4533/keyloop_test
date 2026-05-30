package domain

import "time"

// BayStatus represents the operational status of a service bay.
type BayStatus string

const (
	BayStatusActive      BayStatus = "active"
	BayStatusInactive    BayStatus = "inactive"
	BayStatusMaintenance BayStatus = "maintenance"
)

// ServiceBay represents a physical service bay at a dealership where
// vehicles are serviced. A bay can only handle one appointment at a time.
type ServiceBay struct {
	ID           string
	DealershipID string
	Name         string
	BayNumber    int
	Status       BayStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsAvailable returns true if the bay is operationally active.
func (sb *ServiceBay) IsAvailable() bool {
	return sb.Status == BayStatusActive
}
