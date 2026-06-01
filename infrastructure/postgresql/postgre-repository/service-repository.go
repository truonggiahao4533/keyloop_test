package postgrerepository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
)

type serviceRepo struct {
	q *sqlc.Queries
}

func NewServiceRepository(db *sql.DB) repoIntf.ServiceRepository {
	return &serviceRepo{q: sqlc.New(db)}
}

func toServiceDomain(s sqlc.Service) (*domain.Service, error) {
	price, err := strconv.ParseFloat(s.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price %q: %w", s.Price, err)
	}
	return &domain.Service{
		ID:               s.ID.String(),
		Name:             s.Name,
		Description:      s.Description,
		EstimatedMinutes: int(s.EstimatedMinutes),
		Price:            price,
		IsActive:         s.IsActive,
		DeletedAt:        s.DeletedAt.Time,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}, nil
}

func (r *serviceRepo) GetService(ctx context.Context, id string) (*domain.Service, error) {
	row, err := r.q.GetService(ctx, uuid.MustParse(id))
	if err != nil {
		return nil, err
	}
	return toServiceDomain(row)
}

func (r *serviceRepo) ListServices(ctx context.Context) ([]*domain.Service, error) {
	rows, err := r.q.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*domain.Service, 0, len(rows))
	for _, row := range rows {
		svc, err := toServiceDomain(row)
		if err != nil {
			return nil, err
		}
		res = append(res, svc)
	}
	return res, nil
}

func (r *serviceRepo) CreateService(ctx context.Context, s *domain.Service) error {
	uid, _ := uuid.Parse(s.ID)
	created, err := r.q.CreateService(ctx, sqlc.CreateServiceParams{
		ID:               uid,
		Name:             s.Name,
		Description:      s.Description,
		EstimatedMinutes: int32(s.EstimatedMinutes),
		Price:            strconv.FormatFloat(s.Price, 'f', -1, 64),
		IsActive:         s.IsActive,
	})
	if err != nil {
		return err
	}
	s.CreatedAt = created.CreatedAt
	s.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *serviceRepo) UpdateService(ctx context.Context, s *domain.Service) error {
	uid, _ := uuid.Parse(s.ID)
	updated, err := r.q.UpdateService(ctx, sqlc.UpdateServiceParams{
		ID:               uid,
		Name:             s.Name,
		Description:      s.Description,
		EstimatedMinutes: int32(s.EstimatedMinutes),
		Price:            strconv.FormatFloat(s.Price, 'f', -1, 64),
		IsActive:         s.IsActive,
	})
	if err != nil {
		return err
	}
	s.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *serviceRepo) DeleteService(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteService(ctx, uid)
}
