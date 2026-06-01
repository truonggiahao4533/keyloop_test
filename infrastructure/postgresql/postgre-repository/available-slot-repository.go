package postgrerepository

import (
	"context"
	"database/sql"
	"time"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
)

type AvailableSlotRepository struct {
	q *sqlc.Queries
}

func NewAvailabilitySlotRepository(db *sql.DB) repoIntf.AvailabilitySlotRepository {
	return &AvailableSlotRepository{q: sqlc.New(db)}
}

func (r *AvailableSlotRepository) GetAvailableSlots(ctx context.Context, startTime, endTime time.Time, dealershipID string, duration time.Duration, services []string) ([]domain.AvailableSlot, error) {
	input := sqlc.GetAvailableSlotsParams{
		StartDatetime: startTime,
		EndDatetime:   endTime,
		DealershipID:  uuid.Must(uuid.Parse(dealershipID)),
		ServiceTypes:  services,
		Duration:      int64(duration.Seconds()),
	}
	starts, err := r.q.GetAvailableSlots(ctx, input)
	if err != nil {
		return nil, err
	}
	slots := make([]domain.AvailableSlot, len(starts))
	for i, s := range starts {
		slots[i] = domain.AvailableSlot{Start: s, End: s.Add(duration)}
	}
	return slots, nil
}
