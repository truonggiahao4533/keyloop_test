package domain

import "time"

// Vehicle represents a customer's vehicle that needs servicing.
type Vehicle struct {
	ID           string
	CustomerID   string
	Make         string
	Model        string
	Year         int
	VIN          string
	LicensePlate string
	DeletedAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DisplayName returns a human-readable identifier for the vehicle.
func (v *Vehicle) DisplayName() string {
	return v.Make + " " + v.Model
}
