package domain

// AvailableResources is the result of a pre-booking availability check:
// one technician (qualified for all requested services and free during the
// slot) paired with one service bay (active and free during the same slot).
type AvailableResources struct {
	TechnicianID string
	BayID        string
}
