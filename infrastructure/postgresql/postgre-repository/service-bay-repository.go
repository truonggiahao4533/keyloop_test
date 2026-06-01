package postgrerepository

import (
	"context"
	"database/sql"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
)

type serviceBayRepo struct {
	q *sqlc.Queries
}

func NewServiceBayRepository(db *sql.DB) repoIntf.ServiceBayRepository {
	return &serviceBayRepo{q: sqlc.New(db)}
}

func (r *serviceBayRepo) GetServiceBay(ctx context.Context, id string) (*domain.ServiceBay, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	b, err := r.q.GetServiceBay(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &domain.ServiceBay{
		ID:           b.ID.String(),
		DealershipID: b.DealershipID.String(),
		Name:         b.Name,
		BayNumber:    int(b.BayNumber),
		Status:       domain.BayStatus(b.Status),
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}, nil
}

func (r *serviceBayRepo) ListServiceBaysByDealership(ctx context.Context, dealershipID string) ([]*domain.ServiceBay, error) {
	uid, _ := uuid.Parse(dealershipID)
	bs, err := r.q.ListServiceBaysByDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]*domain.ServiceBay, len(bs))
	for i, b := range bs {
		res[i] = &domain.ServiceBay{
			ID:           b.ID.String(),
			DealershipID: b.DealershipID.String(),
			Name:         b.Name,
			BayNumber:    int(b.BayNumber),
			Status:       domain.BayStatus(b.Status),
			CreatedAt:    b.CreatedAt,
			UpdatedAt:    b.UpdatedAt,
		}
	}
	return res, nil
}

func (r *serviceBayRepo) CreateServiceBay(ctx context.Context, b *domain.ServiceBay) error {
	uid, _ := uuid.Parse(b.ID)
	duid, _ := uuid.Parse(b.DealershipID)
	created, err := r.q.CreateServiceBay(ctx, sqlc.CreateServiceBayParams{
		ID:           uid,
		DealershipID: duid,
		Name:         b.Name,
		BayNumber:    int32(b.BayNumber),
		Status:       sqlc.BayStatus(b.Status),
	})
	if err != nil {
		return err
	}
	b.CreatedAt = created.CreatedAt
	b.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *serviceBayRepo) UpdateServiceBay(ctx context.Context, b *domain.ServiceBay) error {
	uid, _ := uuid.Parse(b.ID)
	updated, err := r.q.UpdateServiceBay(ctx, sqlc.UpdateServiceBayParams{
		ID:        uid,
		Name:      b.Name,
		BayNumber: int32(b.BayNumber),
		Status:    sqlc.BayStatus(b.Status),
	})
	if err != nil {
		return err
	}
	b.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *serviceBayRepo) DeleteServiceBay(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteServiceBay(ctx, uid)
}
