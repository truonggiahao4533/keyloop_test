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

type technicianRepo struct {
	q *sqlc.Queries
}

func NewTechnicianRepository(db *sql.DB) repoIntf.TechnicianRepository {
	return &technicianRepo{q: sqlc.New(db)}
}

func (r *technicianRepo) GetTechnician(ctx context.Context, id string) (*domain.Technician, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	t, err := r.q.GetTechnician(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &domain.Technician{
		ID:           t.ID.String(),
		DealershipID: t.DealershipID.String(),
		FirstName:    t.FirstName,
		LastName:     t.LastName,
		Status:       domain.TechnicianStatus(t.Status),
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}, nil
}

func (r *technicianRepo) ListTechniciansByDealership(ctx context.Context, dealershipID string) ([]*domain.Technician, error) {
	uid, _ := uuid.Parse(dealershipID)
	ts, err := r.q.ListTechniciansByDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]*domain.Technician, len(ts))
	for i, t := range ts {
		res[i] = &domain.Technician{
			ID:           t.ID.String(),
			DealershipID: t.DealershipID.String(),
			FirstName:    t.FirstName,
			LastName:     t.LastName,
			Status:       domain.TechnicianStatus(t.Status),
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		}
	}
	return res, nil
}

func (r *technicianRepo) CreateTechnician(ctx context.Context, t *domain.Technician) error {
	uid, _ := uuid.Parse(t.ID)
	duid, _ := uuid.Parse(t.DealershipID)
	created, err := r.q.CreateTechnician(ctx, sqlc.CreateTechnicianParams{
		ID:           uid,
		DealershipID: duid,
		FirstName:    t.FirstName,
		LastName:     t.LastName,
		Status:       sqlc.TechnicianStatus(t.Status),
	})
	if err != nil {
		return err
	}
	t.CreatedAt = created.CreatedAt
	t.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *technicianRepo) UpdateTechnician(ctx context.Context, t *domain.Technician) error {
	uid, _ := uuid.Parse(t.ID)
	updated, err := r.q.UpdateTechnician(ctx, sqlc.UpdateTechnicianParams{
		ID:        uid,
		FirstName: t.FirstName,
		LastName:  t.LastName,
		Status:    sqlc.TechnicianStatus(t.Status),
	})
	if err != nil {
		return err
	}
	t.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *technicianRepo) GetAvailableTechnicians(ctx context.Context, dealershipID string, start, end time.Time, serviceIDs []string) ([]*domain.Technician, error) {
	did, err := uuid.Parse(dealershipID)
	if err != nil {
		return nil, err
	}
	ts, err := r.q.GetAvailableTechnicians(ctx, sqlc.GetAvailableTechniciansParams{
		DealershipID: did,
		StartTime:    start,
		EndTime:      end,
		ServiceIds:   serviceIDs,
	})
	if err != nil {
		return nil, err
	}
	res := make([]*domain.Technician, len(ts))
	for i, t := range ts {
		res[i] = &domain.Technician{
			ID:           t.ID.String(),
			DealershipID: t.DealershipID.String(),
			FirstName:    t.FirstName,
			LastName:     t.LastName,
			Status:       domain.TechnicianStatus(t.Status),
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		}
	}
	return res, nil
}

func (r *technicianRepo) DeleteTechnician(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteTechnician(ctx, uid)
}
