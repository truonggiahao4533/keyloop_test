package httpHandler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"keyloop-test/internal/usecase"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type AppointmentHandler struct {
	uc *usecase.AppoinmentBookingUseCase
}

func NewAppointmentHandler(uc *usecase.AppoinmentBookingUseCase) *AppointmentHandler {
	return &AppointmentHandler{uc: uc}
}

func (h *AppointmentHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/appointments", h.BookAppointment)
		v1.GET("/appointments/available-slots", h.GetAvailableSlots)
	}
}

// Request / response types

type bookAppointmentRequest struct {
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
// @Failure      422   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /api/v1/appointments [post]
func (h *AppointmentHandler) BookAppointment(c *gin.Context) {
	var req bookAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mockCustomerID := "c1a2b3c4-d5e6-7890-abcd-ef1234567890" //TODO: Get from auth context

	out, err := h.uc.BookAppointment(c.Request.Context(), &usecase.AppoinmentBookingInput{
		DealershipID:     req.DealershipID,
		VehicleID:        req.VehicleID,
		Services:         req.Services,
		DesiredStartTime: req.DesiredStartTime,
		CustomerID:       mockCustomerID,
	})
	if err != nil {
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

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
// @Param        vehicle_id             query     string    false  "Vehicle UUID"
// @Param        customer_id            query     string    false  "Customer UUID"
// @Param        services               query     []string  false  "Service type identifiers (repeatable)"  collectionFormat(multi)
// @Param        total_duration_minutes query     int       true   "Total service duration in minutes"
// @Param        desired_time           query     string    true   "Desired start datetime (RFC3339)"
// @Success      200  {object}  availableSlotsResponse
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      422  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /api/v1/appointments/available-slots [get]
func (h *AppointmentHandler) GetAvailableSlots(c *gin.Context) {
	dealershipID := c.Query("dealership_id")
	vehicleID := c.Query("vehicle_id")
	customerID := c.Query("customer_id")
	serviceTypes := c.QueryArray("services")
	desiredTimeStr := c.Query("desired_time")
	totalDurStr := c.Query("total_duration_minutes")

	if dealershipID == "" || desiredTimeStr == "" || totalDurStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dealership_id, desired_time, and total_duration_minutes are required"})
		return
	}

	desiredTime, err := time.Parse(time.RFC3339, desiredTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "desired_time must be RFC3339 format"})
		return
	}

	var totalMinutes int
	if _, err := fmt.Sscanf(totalDurStr, "%d", &totalMinutes); err != nil || totalMinutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "total_duration_minutes must be a positive integer"})
		return
	}

	out, err := h.uc.AvailableSlots(c.Request.Context(), &usecase.AvailableSlotsInput{
		CustomerID:    customerID,
		DealershipID:  dealershipID,
		Services:      serviceTypes,
		TotalDuration: time.Duration(totalMinutes) * time.Minute,
		VehicleID:     vehicleID,
		DesiredDate:   desiredTime,
	})
	if err != nil {
		c.JSON(appointmentErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

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

func appointmentErrorStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrDealershipNotFound),
		errors.Is(err, usecase.ErrVehicleNotFound):
		return http.StatusNotFound
	case errors.Is(err, usecase.ErrDesiredDateInPast),
		errors.Is(err, usecase.ErrStartTimeInPast):
		return http.StatusUnprocessableEntity
	case errors.Is(err, usecase.ErrDealershipIDRequired),
		errors.Is(err, usecase.ErrServiceTypesRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
