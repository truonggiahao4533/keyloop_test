package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"
)

// --- Dealership Repository ---
type dealershipRepo struct {
	q *sqlc.Queries
}

func NewDealershipRepository(db *sql.DB) repoIntf.DealershipRepository {
	return &dealershipRepo{q: sqlc.New(db)}
}

func (r *dealershipRepo) GetDealership(ctx context.Context, id string) (domain.Dealership, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.Dealership{}, err
	}
	d, err := r.q.GetDealership(ctx, uid)
	if err != nil {
		return domain.Dealership{}, err
	}
	return domain.Dealership{
		ID:        d.ID.String(),
		Name:      d.Name,
		Address:   d.Address,
		City:      d.City,
		Phone:     d.Phone,
		IsActive:  d.IsActive,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}, nil
}

func (r *dealershipRepo) ListDealerships(ctx context.Context) ([]domain.Dealership, error) {
	ds, err := r.q.ListDealerships(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Dealership, len(ds))
	for i, d := range ds {
		res[i] = domain.Dealership{
			ID:        d.ID.String(),
			Name:      d.Name,
			Address:   d.Address,
			City:      d.City,
			Phone:     d.Phone,
			IsActive:  d.IsActive,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return res, nil
}

func (r *dealershipRepo) CreateDealership(ctx context.Context, d *domain.Dealership) error {
	uid, _ := uuid.Parse(d.ID)
	created, err := r.q.CreateDealership(ctx, sqlc.CreateDealershipParams{
		ID:       uid,
		Name:     d.Name,
		Address:  d.Address,
		City:     d.City,
		Phone:    d.Phone,
		IsActive: d.IsActive,
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
		ID:       uid,
		Name:     d.Name,
		Address:  d.Address,
		City:     d.City,
		Phone:    d.Phone,
		IsActive: d.IsActive,
	})
	if err != nil {
		return err
	}
	d.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *dealershipRepo) DeleteDealership(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteDealership(ctx, uid)
}

// --- Service Bay Repository ---
type serviceBayRepo struct {
	q *sqlc.Queries
}

func NewServiceBayRepository(db *sql.DB) repoIntf.ServiceBayRepository {
	return &serviceBayRepo{q: sqlc.New(db)}
}

func (r *serviceBayRepo) GetServiceBay(ctx context.Context, id string) (domain.ServiceBay, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ServiceBay{}, err
	}
	b, err := r.q.GetServiceBay(ctx, uid)
	if err != nil {
		return domain.ServiceBay{}, err
	}
	return domain.ServiceBay{
		ID:           b.ID.String(),
		DealershipID: b.DealershipID.String(),
		Name:         b.Name,
		BayNumber:    int(b.BayNumber),
		Status:       domain.BayStatus(b.Status),
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}, nil
}

func (r *serviceBayRepo) ListServiceBaysByDealership(ctx context.Context, dealershipID string) ([]domain.ServiceBay, error) {
	uid, _ := uuid.Parse(dealershipID)
	bs, err := r.q.ListServiceBaysByDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]domain.ServiceBay, len(bs))
	for i, b := range bs {
		res[i] = domain.ServiceBay{
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

// --- Technician Repository ---
type technicianRepo struct {
	q *sqlc.Queries
}

func NewTechnicianRepository(db *sql.DB) repoIntf.TechnicianRepository {
	return &technicianRepo{q: sqlc.New(db)}
}

func (r *technicianRepo) GetTechnician(ctx context.Context, id string) (domain.Technician, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.Technician{}, err
	}
	t, err := r.q.GetTechnician(ctx, uid)
	if err != nil {
		return domain.Technician{}, err
	}
	return domain.Technician{
		ID:           t.ID.String(),
		DealershipID: t.DealershipID.String(),
		FirstName:    t.FirstName,
		LastName:     t.LastName,
		Status:       domain.TechnicianStatus(t.Status),
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}, nil
}

func (r *technicianRepo) ListTechniciansByDealership(ctx context.Context, dealershipID string) ([]domain.Technician, error) {
	uid, _ := uuid.Parse(dealershipID)
	ts, err := r.q.ListTechniciansByDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Technician, len(ts))
	for i, t := range ts {
		res[i] = domain.Technician{
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

func (r *technicianRepo) DeleteTechnician(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteTechnician(ctx, uid)
}

// --- Reservation Repository ---
type reservationRepo struct {
	q *sqlc.Queries
}

func NewReservationRepository(db *sql.DB) repoIntf.ReservationRepository {
	return &reservationRepo{q: sqlc.New(db)}
}

func (r *reservationRepo) GetReservation(ctx context.Context, id string) (domain.Reservation, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.Reservation{}, err
	}
	res, err := r.q.GetReservation(ctx, uid)
	if err != nil {
		return domain.Reservation{}, err
	}
	status := ""
	if res.Status.Valid {
		status = res.Status.String
	}
	return domain.Reservation{
		ID:           res.ID.String(),
		BayID:        res.BayID.String(),
		TechnicianID: res.TechnicianID.String(),
		StartTime:    res.StartTime,
		EndTime:      res.EndTime,
		UserID:       res.UserID.String(),
		ExpiresAt:    res.ExpiresAt,
		Status:       status,
	}, nil
}

func (r *reservationRepo) ListReservationsByDealership(ctx context.Context, dealershipID string) ([]domain.Reservation, error) {
	uid, _ := uuid.Parse(dealershipID)
	rs, err := r.q.ListReservationsByDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Reservation, len(rs))
	for i, res := range rs {
		status := ""
		if res.Status.Valid {
			status = res.Status.String
		}
		out[i] = domain.Reservation{
			ID:           res.ID.String(),
			BayID:        res.BayID.String(),
			TechnicianID: res.TechnicianID.String(),
			StartTime:    res.StartTime,
			EndTime:      res.EndTime,
			UserID:       res.UserID.String(),
			ExpiresAt:    res.ExpiresAt,
			Status:       status,
		}
	}
	return out, nil
}

func (r *reservationRepo) CreateReservation(ctx context.Context, res *domain.Reservation) error {
	uid, _ := uuid.Parse(res.ID)
	buid, _ := uuid.Parse(res.BayID)
	tuid, _ := uuid.Parse(res.TechnicianID)
	uuid_, _ := uuid.Parse(res.UserID)

	created, err := r.q.CreateReservation(ctx, sqlc.CreateReservationParams{
		ID:           uid,
		BayID:        buid,
		TechnicianID: tuid,
		StartTime:    res.StartTime,
		EndTime:      res.EndTime,
		UserID:       uuid_,
		ExpiresAt:    res.ExpiresAt,
		Status:       sql.NullString{String: res.Status, Valid: res.Status != ""},
	})
	if err != nil {
		return err
	}
	res.ID = created.ID.String()
	return nil
}

func (r *reservationRepo) UpdateReservationStatus(ctx context.Context, id string, status string) error {
	uid, _ := uuid.Parse(id)
	_, err := r.q.UpdateReservationStatus(ctx, sqlc.UpdateReservationStatusParams{
		ID:     uid,
		Status: sql.NullString{String: status, Valid: status != ""},
	})
	return err
}

func (r *reservationRepo) DeleteExpiredReservations(ctx context.Context) error {
	return r.q.DeleteExpiredReservations(ctx)
}

func (r *reservationRepo) DeleteReservation(ctx context.Context, id string) error {
	uid, _ := uuid.Parse(id)
	return r.q.DeleteReservation(ctx, uid)
}
