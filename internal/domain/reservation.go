package domain

import "time"

type Reservation struct {
	ID           string
	BayID        string
	TechnicianID string
	StartTime    time.Time
	EndTime      time.Time
	UserID       string
	ExpiresAt    time.Time
	Status       string
}
