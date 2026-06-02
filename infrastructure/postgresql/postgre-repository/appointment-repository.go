package postgrerepository

import (
	"context"
	"database/sql"
	"encoding/json"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const pgExclusionViolation = "23P01"

type appointmentRepo struct {
	q *sqlc.Queries
}

func NewAppointmentRepository(db *sql.DB) repoIntf.AppointmentRepository {
	return &appointmentRepo{q: sqlc.New(db)}
}

func (r *appointmentRepo) GetAppointment(ctx context.Context, id string) (*domain.Appointment, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	a, err := r.q.GetAppointment(ctx, uid)
	if err != nil {
		return nil, err
	}
	var services []*domain.ServiceSnapshot
	if err = json.Unmarshal(a.Services, &services); err != nil {
		return nil, err
	}
	return &domain.Appointment{
		ID:           a.ID.String(),
		CustomerID:   a.CustomerID.String(),
		VehicleID:    a.VehicleID.String(),
		DealershipID: a.DealershipID.String(),
		ServiceBayID: a.ServiceBayID.String(),
		TechnicianID: a.TechnicianID.String(),
		Services:     services,
		Status:       domain.AppointmentStatus(a.Status),
		StartTime:    a.StartTime,
		EndTime:      a.EndTime,
		Notes:        a.Notes.String,
		DeletedAt:    a.DeletedAt.Time,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}, nil
}

func (r *appointmentRepo) ListAppointmentsByDealership(ctx context.Context, dealershipID string) ([]domain.Appointment, error) {
	uid, err := uuid.Parse(dealershipID)
	if err != nil {
		return nil, err
	}
	as, err := r.q.ListAppointmentsByDealership(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Appointment, len(as))
	for i, a := range as {
		var services []*domain.ServiceSnapshot
		if err := json.Unmarshal(a.Services, &services); err != nil {
			return nil, err
		}
		res[i] = domain.Appointment{
			ID:           a.ID.String(),
			CustomerID:   a.CustomerID.String(),
			VehicleID:    a.VehicleID.String(),
			DealershipID: a.DealershipID.String(),
			ServiceBayID: a.ServiceBayID.String(),
			TechnicianID: a.TechnicianID.String(),
			Services:     services,
			Status:       domain.AppointmentStatus(a.Status),
			StartTime:    a.StartTime,
			EndTime:      a.EndTime,
			Notes:        a.Notes.String,
			DeletedAt:    a.DeletedAt.Time,
			CreatedAt:    a.CreatedAt,
			UpdatedAt:    a.UpdatedAt,
		}
	}
	return res, nil
}

func (r *appointmentRepo) ListAppointmentsByCustomer(ctx context.Context, customerID string) ([]domain.Appointment, error) {
	uid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	as, err := r.q.ListAppointmentsByCustomer(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Appointment, len(as))
	for i, a := range as {
		var services []*domain.ServiceSnapshot
		if err := json.Unmarshal(a.Services, &services); err != nil {
			return nil, err
		}
		res[i] = domain.Appointment{
			ID:           a.ID.String(),
			CustomerID:   a.CustomerID.String(),
			VehicleID:    a.VehicleID.String(),
			DealershipID: a.DealershipID.String(),
			ServiceBayID: a.ServiceBayID.String(),
			TechnicianID: a.TechnicianID.String(),
			Services:     services,
			Status:       domain.AppointmentStatus(a.Status),
			StartTime:    a.StartTime,
			EndTime:      a.EndTime,
			Notes:        a.Notes.String,
			DeletedAt:    a.DeletedAt.Time,
			CreatedAt:    a.CreatedAt,
			UpdatedAt:    a.UpdatedAt,
		}
	}
	return res, nil
}

func (r *appointmentRepo) ListAppointmentsByCustomerAndDealership(ctx context.Context, customerID, dealershipID string) ([]domain.Appointment, error) {
	cuid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	duid, err := uuid.Parse(dealershipID)
	if err != nil {
		return nil, err
	}
	as, err := r.q.ListAppointmentsByCustomerAndDealership(ctx, sqlc.ListAppointmentsByCustomerAndDealershipParams{
		CustomerID:   cuid,
		DealershipID: duid,
	})
	if err != nil {
		return nil, err
	}
	res := make([]domain.Appointment, len(as))
	for i, a := range as {
		var services []*domain.ServiceSnapshot
		if err := json.Unmarshal(a.Services, &services); err != nil {
			return nil, err
		}
		res[i] = domain.Appointment{
			ID:           a.ID.String(),
			CustomerID:   a.CustomerID.String(),
			VehicleID:    a.VehicleID.String(),
			DealershipID: a.DealershipID.String(),
			ServiceBayID: a.ServiceBayID.String(),
			TechnicianID: a.TechnicianID.String(),
			Services:     services,
			Status:       domain.AppointmentStatus(a.Status),
			StartTime:    a.StartTime,
			EndTime:      a.EndTime,
			Notes:        a.Notes.String,
			DeletedAt:    a.DeletedAt.Time,
			CreatedAt:    a.CreatedAt,
			UpdatedAt:    a.UpdatedAt,
		}
	}
	return res, nil
}

func (r *appointmentRepo) CreateAppointment(ctx context.Context, appt *domain.Appointment) (*domain.Appointment, error) {
	cuid, err := uuid.Parse(appt.CustomerID)
	if err != nil {
		return nil, err
	}
	vid, err := uuid.Parse(appt.VehicleID)
	if err != nil {
		return nil, err
	}
	did, err := uuid.Parse(appt.DealershipID)
	if err != nil {
		return nil, err
	}

	var services []*domain.ServiceSnapshot
	for _, s := range appt.Services {
		services = append(services, &domain.ServiceSnapshot{
			ServiceID:        s.ServiceID,
			Name:             s.Name,
			EstimatedMinutes: s.EstimatedMinutes,
			Price:            s.Price,
		})
	}

	servicesJSON, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, len(services))
	for i, s := range services {
		serviceIDs[i] = s.ServiceID
	}

	result, err := r.q.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		CustomerID:   cuid,
		VehicleID:    vid,
		DealershipID: did,
		Services:     servicesJSON,
		StartTime:    appt.StartTime,
		EndTime:      appt.EndTime,
		Status:       sqlc.AppointmentStatus(appt.Status),
		Notes:        sql.NullString{String: appt.Notes, Valid: appt.Notes != ""},
		ServiceIds:   serviceIDs,
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == pgExclusionViolation {
			return nil, domain.ErrTimeSlotConflict
		}
		return nil, err
	}
	return &domain.Appointment{
		ID:           result.ID.String(),
		CustomerID:   result.CustomerID.String(),
		VehicleID:    result.VehicleID.String(),
		DealershipID: result.DealershipID.String(),
		ServiceBayID: result.ServiceBayID.String(),
		TechnicianID: result.TechnicianID.String(),
		Services:     services,
		Status:       domain.AppointmentStatus(result.Status),
		StartTime:    result.StartTime,
		EndTime:      result.EndTime,
		Notes:        result.Notes.String,
		DeletedAt:    result.DeletedAt.Time,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	}, nil
}

func (r *appointmentRepo) UpdateAppointmentStatus(ctx context.Context, appt *domain.Appointment, status domain.AppointmentStatus) error {
	uid, err := uuid.Parse(appt.ID)
	if err != nil {
		return err
	}
	_, err = r.q.UpdateAppointmentStatus(ctx, sqlc.UpdateAppointmentStatusParams{
		ID:     uid,
		Status: sqlc.AppointmentStatus(status),
	})
	return err
}

func (r *appointmentRepo) UpdateAppointment(ctx context.Context, id string, status domain.AppointmentStatus, notes string) (*domain.Appointment, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	a, err := r.q.UpdateAppointment(ctx, sqlc.UpdateAppointmentParams{
		ID:     uid,
		Status: sqlc.AppointmentStatus(status),
		Notes:  sql.NullString{String: notes, Valid: notes != ""},
	})
	if err != nil {
		return nil, err
	}
	var services []*domain.ServiceSnapshot
	if err = json.Unmarshal(a.Services, &services); err != nil {
		return nil, err
	}
	return &domain.Appointment{
		ID:           a.ID.String(),
		CustomerID:   a.CustomerID.String(),
		VehicleID:    a.VehicleID.String(),
		DealershipID: a.DealershipID.String(),
		ServiceBayID: a.ServiceBayID.String(),
		TechnicianID: a.TechnicianID.String(),
		Services:     services,
		Status:       domain.AppointmentStatus(a.Status),
		StartTime:    a.StartTime,
		EndTime:      a.EndTime,
		Notes:        a.Notes.String,
		DeletedAt:    a.DeletedAt.Time,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}, nil
}

func (r *appointmentRepo) DeleteAppointment(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteAppointment(ctx, uid)
}
