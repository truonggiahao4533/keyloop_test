package domain

import "time"

// Customer represents a customer who can book service appointments.
type Customer struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Phone     string
	DeletedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FullName returns the customer's full display name.
func (c *Customer) FullName() string {
	return c.FirstName + " " + c.LastName
}
