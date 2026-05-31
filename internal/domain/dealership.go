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
	OpenTime  time.Duration // offset from midnight, e.g. 9*time.Hour = 09:00
	CloseTime time.Duration // offset from midnight, e.g. 17*time.Hour = 17:00
	DeletedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (d *Dealership) IsSlotWithinHours(start, end time.Time) bool {
	startTOD := timeOfDay(start)
	endTOD := timeOfDay(end)
	return startTOD >= d.OpenTime && endTOD <= d.CloseTime
}

func timeOfDay(t time.Time) time.Duration {
	return time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second
}
