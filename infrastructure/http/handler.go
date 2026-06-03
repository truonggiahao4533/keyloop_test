package httpHandler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"keyloop-test/internal/domain"
	"keyloop-test/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// requireUUID trims s, rejects blank values, and validates UUID format.
// It writes the appropriate 400 response and returns ("", false) on failure.
func requireUUID(c *gin.Context, field, value string) (string, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": field + " is required"})
		return "", false
	}
	if !isValidUUID(v) {
		c.JSON(http.StatusBadRequest, gin.H{"error": field + " must be a valid UUID"})
		return "", false
	}
	return v, true
}

type AppointmentHandler struct {
	uc     *usecase.AppoinmentBookingUseCase
	tracer trace.Tracer
}

func NewAppointmentHandler(uc *usecase.AppoinmentBookingUseCase) *AppointmentHandler {
	return &AppointmentHandler{
		uc:     uc,
		tracer: otel.Tracer("keyloop-test/handler"),
	}
}

// Request / response types

type bookAppointmentRequest struct {
	CustomerID       string    `json:"customer_id"    binding:"required" example:"c1a2b3c4-d5e6-7890-abcd-ef1234567890"`
	DealershipID     string    `json:"dealership_id"  binding:"required" example:"d1e2f3a4-b5c6-7890-abcd-ef1234567890"`
	VehicleID        string    `json:"vehicle_id"     binding:"required" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	Services         []string  `json:"services"       binding:"required,min=1" example:"oil_change,tire_rotation"`
	DesiredStartTime time.Time `json:"desired_start_time" binding:"required" example:"2025-01-15T09:00:00Z"`
}

type bookAppointmentResponse struct {
	AppointmentID   string `json:"appointment_id"   example:"f1a2b3c4-d5e6-7890-abcd-ef1234567890"`
	DurationMinutes int    `json:"duration_minutes" example:"75"`
	Message         string `json:"message"          example:"Appointment booked successfully"`
}

type slotJSON struct {
	Start string `json:"start" example:"2025-01-15T09:00:00Z"`
	End   string `json:"end"   example:"2025-01-15T10:00:00Z"`
}

type availableSlotsResponse struct {
	AvailableSlots []slotJSON `json:"available_slots"`
}

type errorResponse struct {
	Error string `json:"error" example:"dealership not found"`
}

// BookAppointment godoc
//
// @Summary      Book an appointment
// @Description  Create a new service appointment at a dealership for the chosen time slot
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Param        body  body      bookAppointmentRequest   true  "Appointment details"
// @Success      201   {object}  bookAppointmentResponse
// @Failure      400   {object}  errorResponse
// @Failure      404   {object}  errorResponse
// @Failure      409   {object}  errorResponse
// @Failure      422   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /api/v1/appointments [post]
func (h *AppointmentHandler) BookAppointment(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.BookAppointment",
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
		),
	)
	defer func() {
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(c.Writer.Status()))
		span.End()
	}()

	var req bookAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	customerID, ok := requireUUID(c, "customer_id", req.CustomerID)
	if !ok {
		return
	}
	dealershipID, ok := requireUUID(c, "dealership_id", req.DealershipID)
	if !ok {
		return
	}
	vehicleID, ok := requireUUID(c, "vehicle_id", req.VehicleID)
	if !ok {
		return
	}
	serviceIDs := make([]string, 0, len(req.Services))
	for _, svcID := range req.Services {
		id, ok := requireUUID(c, "service id", svcID)
		if !ok {
			return
		}
		serviceIDs = append(serviceIDs, id)
	}
	slog.InfoContext(ctx, "booking appointment",
		"dealership_id", dealershipID,
		"vehicle_id", vehicleID,
		"desired_start_time", req.DesiredStartTime,
		"service_count", len(serviceIDs),
	)
	out, err := h.uc.BookAppointment(ctx, &usecase.AppoinmentBookingInput{
		CustomerID:       customerID, //TODO: Replace with JWT token claim
		DealershipID:     dealershipID,
		VehicleID:        vehicleID,
		Services:         serviceIDs,
		DesiredStartTime: req.DesiredStartTime,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "book appointment failed", "error", err)
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(ctx, "appointment booked",
		"appointment_id", out.AppointmentID,
		"duration_minutes", out.DurationMinutes,
	)
	c.JSON(http.StatusCreated, bookAppointmentResponse{
		AppointmentID:   out.AppointmentID,
		DurationMinutes: out.DurationMinutes,
		Message:         out.Message,
	})
}

// GetAvailableSlots godoc
//
// @Summary      List available slots
// @Description  Returns available 30-minute time slots within 7 days of desired_time where a bay and qualified technician are both free
// @Tags         appointments
// @Produce      json
// @Param        dealership_id          query     string    true   "Dealership UUID"
// @Param        vehicle_id             query     string    true  "Vehicle UUID"
// @Param        customer_id            query     string    true  "Customer UUID"
// @Param        services               query     []string  true  "Service type identifiers (repeatable)"  collectionFormat(multi)
// @Param        desired_time           query     string    true   "Desired start datetime (RFC3339)"
// @Success      200  {object}  availableSlotsResponse
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      422  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/appointments/available-slots [get]
func (h *AppointmentHandler) GetAvailableSlots(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.GetAvailableSlots",
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
		),
	)
	defer func() {
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(c.Writer.Status()))
		span.End()
	}()

	dealershipID, ok := requireUUID(c, "dealership_id", c.Query("dealership_id"))
	if !ok {
		return
	}
	vehicleID, ok := requireUUID(c, "vehicle_id", c.Query("vehicle_id"))
	if !ok {
		return
	}
	customerID, ok := requireUUID(c, "customer_id", c.Query("customer_id"))
	if !ok {
		return
	}
	rawServices := c.QueryArray("services")
	if len(rawServices) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "services is required"})
		return
	}
	serviceIDs := make([]string, 0, len(rawServices))
	for _, svcID := range rawServices {
		id, ok := requireUUID(c, "service id", svcID)
		if !ok {
			return
		}
		serviceIDs = append(serviceIDs, id)
	}
	desiredTimeStr := strings.TrimSpace(c.Query("desired_time"))
	if desiredTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "desired_time is required"})
		return
	}
	desiredTime, err := time.Parse(time.RFC3339, desiredTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "desired_time must be RFC3339 format"})
		return
	}

	slog.InfoContext(ctx, "getting available slots",
		"dealership_id", dealershipID,
		"vehicle_id", vehicleID,
		"desired_time", desiredTime,
		"service_count", len(serviceIDs),
	)
	out, err := h.uc.AvailableSlots(ctx, &usecase.AvailableSlotsInput{
		CustomerID:   customerID,
		DealershipID: dealershipID,
		Services:     serviceIDs,
		VehicleID:    vehicleID,
		DesiredDate:  desiredTime,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "get available slots failed", "error", err)
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(ctx, "available slots returned", "count", len(out.AvailableSlots))
	type slot struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	slots := make([]slot, len(out.AvailableSlots))
	for i, s := range out.AvailableSlots {
		slots[i] = slot{Start: s.Start.Format(time.RFC3339), End: s.End.Format(time.RFC3339)}
	}

	c.JSON(http.StatusOK, gin.H{"available_slots": slots})
}

type serviceSnapshotJSON struct {
	ServiceID        string  `json:"service_id"         example:"oil_change"`
	Name             string  `json:"name"               example:"Oil Change"`
	EstimatedMinutes int     `json:"estimated_minutes"  example:"30"`
	Price            float64 `json:"price"              example:"49.99"`
}

type appointmentJSON struct {
	ID           string                `json:"id"            example:"f1a2b3c4-d5e6-7890-abcd-ef1234567890"`
	CustomerID   string                `json:"customer_id"   example:"c1a2b3c4-d5e6-7890-abcd-ef1234567890"`
	VehicleID    string                `json:"vehicle_id"    example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	DealershipID string                `json:"dealership_id" example:"d1e2f3a4-b5c6-7890-abcd-ef1234567890"`
	Services     []serviceSnapshotJSON `json:"services"`
	Status       string                `json:"status"        example:"pending"`
	StartTime    string                `json:"start_time"    example:"2025-01-15T09:00:00Z"`
	EndTime      string                `json:"end_time"      example:"2025-01-15T10:15:00Z"`
	Notes        string                `json:"notes"         example:""`
	CreatedAt    string                `json:"created_at"    example:"2025-01-10T08:00:00Z"`
}

type listAppointmentsResponse struct {
	Appointments []appointmentJSON `json:"appointments"`
}

// ListAppointments godoc
//
// @Summary      List appointments
// @Description  Returns appointments filtered by customer_id or dealership_id (one is required)
// @Tags         appointments
// @Produce      json
// @Param        customer_id    query     string  false  "Customer UUID"
// @Param        dealership_id  query     string  false  "Dealership UUID"
// @Success      200  {object}  listAppointmentsResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/appointments [get]
func (h *AppointmentHandler) ListAppointments(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.ListAppointments",
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
		),
	)
	defer func() {
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(c.Writer.Status()))
		span.End()
	}()

	customerID := strings.TrimSpace(c.Query("customer_id"))
	dealershipID := strings.TrimSpace(c.Query("dealership_id"))

	if customerID == "" && dealershipID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id or dealership_id is required"})
		return
	}
	if customerID != "" && !isValidUUID(customerID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id must be a valid UUID"})
		return
	}
	if dealershipID != "" && !isValidUUID(dealershipID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dealership_id must be a valid UUID"})
		return
	}

	slog.InfoContext(ctx, "listing appointments",
		"customer_id", customerID,
		"dealership_id", dealershipID,
	)
	out, err := h.uc.ListAppointments(ctx, &usecase.ListAppointmentsInput{
		CustomerID:   customerID,
		DealershipID: dealershipID,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "list appointments failed", "error", err)
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	items := make([]appointmentJSON, len(out.Appointments))
	for i, a := range out.Appointments {
		snapshots := make([]serviceSnapshotJSON, len(a.Services))
		for j, s := range a.Services {
			snapshots[j] = serviceSnapshotJSON{
				ServiceID:        s.ServiceID,
				Name:             s.Name,
				EstimatedMinutes: s.EstimatedMinutes,
				Price:            s.Price,
			}
		}
		items[i] = appointmentJSON{
			ID:           a.ID,
			CustomerID:   a.CustomerID,
			VehicleID:    a.VehicleID,
			DealershipID: a.DealershipID,
			Services:     snapshots,
			Status:       a.Status,
			StartTime:    a.StartTime.Format(time.RFC3339),
			EndTime:      a.EndTime.Format(time.RFC3339),
			Notes:        a.Notes,
			CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		}
	}

	slog.InfoContext(ctx, "appointments listed", "count", len(items))
	c.JSON(http.StatusOK, listAppointmentsResponse{Appointments: items})
}

// DeleteAppointment godoc
//
// @Summary      Soft-delete an appointment
// @Description  Cancels and archives an appointment; only pending or confirmed appointments can be deleted
// @Tags         appointments
// @Produce      json
// @Param        id   path      string  true  "Appointment UUID"
// @Success      204
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      422  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/appointments/{id} [delete]
func (h *AppointmentHandler) DeleteAppointment(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.DeleteAppointment",
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
		),
	)
	defer func() {
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(c.Writer.Status()))
		span.End()
	}()

	id, ok := requireUUID(c, "id", c.Param("id"))
	if !ok {
		return
	}
	slog.InfoContext(ctx, "deleting appointment", "appointment_id", id)
	if err := h.uc.SoftDeleteAppointment(ctx, id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "delete appointment failed", "appointment_id", id, "error", err)
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(ctx, "appointment deleted", "appointment_id", id)
	c.Status(http.StatusNoContent)
}

type updateAppointmentRequest struct {
	Status *string `json:"status"`
	Notes  *string `json:"notes"`
}

// UpdateAppointment godoc
//
// @Summary      Update an appointment
// @Description  Update the status and/or notes of an existing appointment; at least one field is required
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "Appointment UUID"
// @Param        body  body      updateAppointmentRequest true  "Fields to update"
// @Success      200   {object}  appointmentJSON
// @Failure      400   {object}  errorResponse
// @Failure      404   {object}  errorResponse
// @Failure      422   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /api/v1/appointments/{id} [patch]
func (h *AppointmentHandler) UpdateAppointment(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "handler.UpdateAppointment",
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
		),
	)
	defer func() {
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(c.Writer.Status()))
		span.End()
	}()

	id, ok := requireUUID(c, "id", c.Param("id"))
	if !ok {
		return
	}
	var req updateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == nil && req.Notes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one of status or notes must be provided"})
		return
	}

	var statusPtr *domain.AppointmentStatus
	if req.Status != nil {
		s := domain.AppointmentStatus(strings.TrimSpace(*req.Status))
		if s == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must not be blank"})
			return
		}
		statusPtr = &s
	}
	var notesPtr *string
	if req.Notes != nil {
		trimmed := strings.TrimSpace(*req.Notes)
		notesPtr = &trimmed
	}

	slog.InfoContext(ctx, "updating appointment", "appointment_id", id)
	out, err := h.uc.UpdateAppointment(ctx, &usecase.UpdateAppointmentInput{
		AppointmentID: id,
		Status:        statusPtr,
		Notes:         notesPtr,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "update appointment failed", "appointment_id", id, "error", err)
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	slog.InfoContext(ctx, "appointment updated", "appointment_id", id, "status", out.Status)

	snapshots := make([]serviceSnapshotJSON, len(out.Services))
	for j, s := range out.Services {
		snapshots[j] = serviceSnapshotJSON{
			ServiceID:        s.ServiceID,
			Name:             s.Name,
			EstimatedMinutes: s.EstimatedMinutes,
			Price:            s.Price,
		}
	}
	c.JSON(http.StatusOK, appointmentJSON{
		ID:           out.ID,
		CustomerID:   out.CustomerID,
		VehicleID:    out.VehicleID,
		DealershipID: out.DealershipID,
		Services:     snapshots,
		Status:       out.Status,
		StartTime:    out.StartTime.Format(time.RFC3339),
		EndTime:      out.EndTime.Format(time.RFC3339),
		Notes:        out.Notes,
		CreatedAt:    out.CreatedAt.Format(time.RFC3339),
	})
}

func appointmentErrorStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrDealershipNotFound),
		errors.Is(err, usecase.ErrVehicleNotFound),
		errors.Is(err, usecase.ErrAppointmentNotFound),
		errors.Is(err, domain.ErrServiceNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrTimeSlotConflict),
		errors.Is(err, domain.ErrNoAvailableResources):
		return http.StatusConflict
	case errors.Is(err, usecase.ErrDesiredDateInPast),
		errors.Is(err, usecase.ErrStartTimeInPast),
		errors.Is(err, usecase.ErrSlotOutsideWorkingHours),
		errors.Is(err, usecase.ErrSlotNotOnWorkingDay),
		errors.Is(err, usecase.ErrInvalidStatusTransition),
		errors.Is(err, usecase.ErrCannotDeleteAppointment):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
