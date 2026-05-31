package postgrerepository

import (
	"context"
	"database/sql"
	"fmt"
	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"
	"strconv"

	"github.com/google/uuid"
)

type serviceDefinitionRepo struct {
	q *sqlc.Queries
}

func NewServiceDefinitionRepository(db *sql.DB) repoIntf.ServiceDefinitionRepository {
	return &serviceDefinitionRepo{q: sqlc.New(db)}
}

func toServiceDomain(s sqlc.ServiceDefinition) (*domain.Service, error) {
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

func (r *serviceDefinitionRepo) GetServiceDefinition(ctx context.Context, id string) (*domain.Service, error) {
	row, err := r.q.GetServiceDefinition(ctx, uuid.MustParse(id))
	if err != nil {
		return nil, err
	}
	return toServiceDomain(row)
}

func (r *serviceDefinitionRepo) ListServiceDefinitions(ctx context.Context) ([]*domain.Service, error) {
	rows, err := r.q.ListServiceDefinitions(ctx)
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

func (r *serviceDefinitionRepo) CreateServiceDefinition(ctx context.Context, s *domain.Service) error {
	uid, _ := uuid.Parse(s.ID)
	created, err := r.q.CreateServiceDefinition(ctx, sqlc.CreateServiceDefinitionParams{
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

func (r *serviceDefinitionRepo) UpdateServiceDefinition(ctx context.Context, s *domain.Service) error {
	uid, _ := uuid.Parse(s.ID)
	updated, err := r.q.UpdateServiceDefinition(ctx, sqlc.UpdateServiceDefinitionParams{
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

func (r *serviceDefinitionRepo) DeleteServiceDefinition(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteServiceDefinition(ctx, uid)
}
