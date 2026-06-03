package postgrerepository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const pgUniqueViolation = "23505"

type unifiedRepo struct {
	q *sqlc.Queries
}

func NewUnifiedRepository(db *sql.DB) repoIntf.BookingRepository {
	return &unifiedRepo{q: sqlc.New(db)}
}

func (r *unifiedRepo) FindAvailableTechnicianAndBay(ctx context.Context, dealershipID string, start, end time.Time, serviceIDs []string) (*domain.AvailableResources, error) {
	did, err := uuid.Parse(dealershipID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.FindAvailableTechnicianAndBay(ctx, sqlc.FindAvailableTechnicianAndBayParams{
		DealershipID: did,
		StartTime:    start,
		EndTime:      end,
		ServiceIds:   serviceIDs,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.AvailableResources{
		TechnicianID: row.TechnicianID.String(),
		BayID:        row.BayID.String(),
	}, nil
}

func (r *unifiedRepo) InsertAppointment(ctx context.Context, appt *domain.Appointment) (*domain.Appointment, error) {
	did, err := uuid.Parse(appt.DealershipID)
	if err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(appt.CustomerID)
	if err != nil {
		return nil, err
	}
	vid, err := uuid.Parse(appt.VehicleID)
	if err != nil {
		return nil, err
	}
	bayID, err := uuid.Parse(appt.ServiceBayID)
	if err != nil {
		return nil, err
	}
	techID, err := uuid.Parse(appt.TechnicianID)
	if err != nil {
		return nil, err
	}

	servicesJSON, err := json.Marshal(appt.Services)
	if err != nil {
		return nil, err
	}

	result, err := r.q.InsertAppointment(ctx, sqlc.InsertAppointmentParams{
		DealershipID: did,
		CustomerID:   cid,
		VehicleID:    vid,
		ServiceBayID: bayID,
		TechnicianID: techID,
		StartTime:    appt.StartTime,
		EndTime:      appt.EndTime,
		Status:       sqlc.AppointmentStatus(appt.Status),
		Notes:        sql.NullString{String: appt.Notes, Valid: appt.Notes != ""},
		Services:     servicesJSON,
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == pgUniqueViolation || pqErr.Code == pgExclusionViolation {
				return nil, domain.ErrDuplicateBooking
			}
		}
		return nil, err
	}

	var services []*domain.ServiceSnapshot
	if err = json.Unmarshal(result.Services, &services); err != nil {
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
