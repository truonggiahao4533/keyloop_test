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

type dealershipRepo struct {
	q *sqlc.Queries
}

func NewDealershipRepository(db *sql.DB) repoIntf.DealershipRepository {
	return &dealershipRepo{q: sqlc.New(db)}
}

func (r *dealershipRepo) GetDealership(ctx context.Context, id string) (*domain.Dealership, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	d, err := r.q.GetDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &domain.Dealership{
		ID:          d.ID.String(),
		Name:        d.Name,
		Address:     d.Address,
		City:        d.City,
		Phone:       d.Phone,
		IsActive:    d.IsActive,
		OpenTime:    timeToDuration(d.OpenTime),
		CloseTime:   timeToDuration(d.CloseTime),
		WorkingDays: toWeekdays(d.WorkingDays),
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}, nil
}

func (r *dealershipRepo) ListDealerships(ctx context.Context) ([]*domain.Dealership, error) {
	ds, err := r.q.ListDealerships(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*domain.Dealership, len(ds))
	for i, d := range ds {
		res[i] = &domain.Dealership{
			ID:          d.ID.String(),
			Name:        d.Name,
			Address:     d.Address,
			City:        d.City,
			Phone:       d.Phone,
			IsActive:    d.IsActive,
			OpenTime:    timeToDuration(d.OpenTime),
			CloseTime:   timeToDuration(d.CloseTime),
			WorkingDays: toWeekdays(d.WorkingDays),
			CreatedAt:   d.CreatedAt,
			UpdatedAt:   d.UpdatedAt,
		}
	}
	return res, nil
}

func (r *dealershipRepo) CreateDealership(ctx context.Context, d *domain.Dealership) error {
	uid, _ := uuid.Parse(d.ID)
	created, err := r.q.CreateDealership(ctx, sqlc.CreateDealershipParams{
		ID:          uid,
		Name:        d.Name,
		Address:     d.Address,
		City:        d.City,
		Phone:       d.Phone,
		IsActive:    d.IsActive,
		OpenTime:    durationToTime(d.OpenTime),
		CloseTime:   durationToTime(d.CloseTime),
		WorkingDays: fromWeekdays(d.WorkingDays),
	})
	if err != nil {
		return err
	}
	d.CreatedAt = created.CreatedAt
	d.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *dealershipRepo) UpdateDealership(ctx context.Context, d *domain.Dealership) error {
	uid, _ := uuid.Parse(d.ID)
	updated, err := r.q.UpdateDealership(ctx, sqlc.UpdateDealershipParams{
		ID:          uid,
		Name:        d.Name,
		Address:     d.Address,
		City:        d.City,
		Phone:       d.Phone,
		IsActive:    d.IsActive,
		OpenTime:    durationToTime(d.OpenTime),
		CloseTime:   durationToTime(d.CloseTime),
		WorkingDays: fromWeekdays(d.WorkingDays),
	})
	if err != nil {
		return err
	}
	d.UpdatedAt = updated.UpdatedAt
	return nil
}

func toWeekdays(days []int32) []time.Weekday {
	result := make([]time.Weekday, len(days))
	for i, d := range days {
		result[i] = time.Weekday(d)
	}
	return result
}

func fromWeekdays(days []time.Weekday) []int32 {
	result := make([]int32, len(days))
	for i, d := range days {
		result[i] = int32(d)
	}
	return result
}

func (r *dealershipRepo) DeleteDealership(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteDealership(ctx, uid)
}
