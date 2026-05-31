package domain

import "time"

type AvailableSlot struct {
	Start time.Time
	End   time.Time
}

// Useful domain methods on the value object
func (s AvailableSlot) Duration() time.Duration {
	return s.End.Sub(s.Start)
}

func (s AvailableSlot) Contains(t time.Time) bool {
	return !t.Before(s.Start) && t.Before(s.End)
}

func (s AvailableSlot) Overlaps(other AvailableSlot) bool {
	return s.Start.Before(other.End) && s.End.After(other.Start)
}
