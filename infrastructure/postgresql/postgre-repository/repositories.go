package postgrerepository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"keyloop-test/infrastructure/postgresql/sqlc"
	"keyloop-test/internal/domain"
	repoIntf "keyloop-test/internal/repository"

	"github.com/google/uuid"
)

// durationToTime converts a time-of-day duration (offset from midnight) to a
// time.Time suitable for Postgres TIME columns (date portion is zeroed).
func durationToTime(d time.Duration) time.Time {
	return time.Date(0, 1, 1, int(d.Hours()), int(d.Minutes())%60, 0, 0, time.UTC)
}

// timeToDuration extracts the time-of-day from a time.Time as a duration from midnight.
func timeToDuration(t time.Time) time.Duration {
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
}

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

// --- Appointment Repository ---
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
	err = json.Unmarshal(a.Services, &services)
	if err != nil {
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
		err := json.Unmarshal(a.Services, &services)
		if err != nil {
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
		err := json.Unmarshal(a.Services, &services)
		if err != nil {
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
	var services []*domain.ServiceSnapshot
	for _, s := range appt.Services {
		services = append(services, &domain.ServiceSnapshot{
			ServiceID:        s.ServiceID,
			Name:             s.Name,
			EstimatedMinutes: s.EstimatedMinutes,
			Price:            s.Price,
		})
	}
	_, err = r.q.UpdateAppointmentStatus(ctx, sqlc.UpdateAppointmentStatusParams{
		ID:     uid,
		Status: sqlc.AppointmentStatus(status),
	})
	return err
}

func (r *appointmentRepo) DeleteAppointment(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteAppointment(ctx, uid)
}

// --- Dealership Repository ---
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
		ID:        d.ID.String(),
		Name:      d.Name,
		Address:   d.Address,
		City:      d.City,
		Phone:     d.Phone,
		IsActive:  d.IsActive,
		OpenTime:  timeToDuration(d.OpenTime),
		CloseTime: timeToDuration(d.CloseTime),
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
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
			ID:        d.ID.String(),
			Name:      d.Name,
			Address:   d.Address,
			City:      d.City,
			Phone:     d.Phone,
			IsActive:  d.IsActive,
			OpenTime:  timeToDuration(d.OpenTime),
			CloseTime: timeToDuration(d.CloseTime),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return res, nil
}

func (r *dealershipRepo) CreateDealership(ctx context.Context, d *domain.Dealership) error {
	uid, _ := uuid.Parse(d.ID)
	created, err := r.q.CreateDealership(ctx, sqlc.CreateDealershipParams{
		ID:        uid,
		Name:      d.Name,
		Address:   d.Address,
		City:      d.City,
		Phone:     d.Phone,
		IsActive:  d.IsActive,
		OpenTime:  durationToTime(d.OpenTime),
		CloseTime: durationToTime(d.CloseTime),
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
		ID:        uid,
		Name:      d.Name,
		Address:   d.Address,
		City:      d.City,
		Phone:     d.Phone,
		IsActive:  d.IsActive,
		OpenTime:  durationToTime(d.OpenTime),
		CloseTime: durationToTime(d.CloseTime),
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

// --- Technician Repository ---
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

func (r *technicianRepo) DeleteTechnician(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteTechnician(ctx, uid)
}

// --- Vehicle Repository ---
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
