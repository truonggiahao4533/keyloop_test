package postgrerepository

import (
	"context"
	"database/sql"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
)

type vehicleRepo struct {
	q *sqlc.Queries
}

func NewVehicleRepository(db *sql.DB) repoIntf.VehicleRepository {
	return &vehicleRepo{q: sqlc.New(db)}
}

func (r *vehicleRepo) GetVehicle(ctx context.Context, id string) (*domain.Vehicle, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	v, err := r.q.GetVehicle(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &domain.Vehicle{
		ID:           v.ID.String(),
		CustomerID:   v.CustomerID.String(),
		Make:         v.Make,
		Model:        v.Model,
		Year:         int(v.Year),
		VIN:          v.Vin,
		LicensePlate: v.LicensePlate,
		DeletedAt:    v.DeletedAt.Time,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}, nil
}

func (r *vehicleRepo) ListVehiclesByCustomer(ctx context.Context, customerID string) ([]domain.Vehicle, error) {
	uid, _ := uuid.Parse(customerID)
	vs, err := r.q.ListVehiclesByCustomer(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Vehicle, len(vs))
	for i, v := range vs {
		res[i] = domain.Vehicle{
			ID:           v.ID.String(),
			CustomerID:   v.CustomerID.String(),
			Make:         v.Make,
			Model:        v.Model,
			Year:         int(v.Year),
			VIN:          v.Vin,
			LicensePlate: v.LicensePlate,
			DeletedAt:    v.DeletedAt.Time,
			CreatedAt:    v.CreatedAt,
			UpdatedAt:    v.UpdatedAt,
		}
	}
	return res, nil
}

func (r *vehicleRepo) CreateVehicle(ctx context.Context, v *domain.Vehicle) error {
	uid, _ := uuid.Parse(v.ID)
	cuid, _ := uuid.Parse(v.CustomerID)
	created, err := r.q.CreateVehicle(ctx, sqlc.CreateVehicleParams{
		ID:           uid,
		CustomerID:   cuid,
		Make:         v.Make,
		Model:        v.Model,
		Year:         int32(v.Year),
		Vin:          v.VIN,
		LicensePlate: v.LicensePlate,
	})
	if err != nil {
		return err
	}
	v.CreatedAt = created.CreatedAt
	v.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *vehicleRepo) UpdateVehicle(ctx context.Context, v *domain.Vehicle) error {
	uid, _ := uuid.Parse(v.ID)
	updated, err := r.q.UpdateVehicle(ctx, sqlc.UpdateVehicleParams{
		ID:           uid,
		Make:         v.Make,
		Model:        v.Model,
		Year:         int32(v.Year),
		Vin:          v.VIN,
		LicensePlate: v.LicensePlate,
	})
	if err != nil {
		return err
	}
	v.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *vehicleRepo) DeleteVehicle(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteVehicle(ctx, uid)
}
