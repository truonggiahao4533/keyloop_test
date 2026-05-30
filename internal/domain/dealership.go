package domain

import "time"

// Dealership represents a service location where appointments take place.
// A dealership owns ServiceBays and employs Technicians.
type Dealership struct {
	ID        string
	Name      string
	Address   string
	City      string
	Phone     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
