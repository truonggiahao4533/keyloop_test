package httpHandler

import (
	_ "keyloop-test/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	appointmentHandler *AppointmentHandler
}

func NewRouter(appointmentHandler *AppointmentHandler) *Router {
	return &Router{appointmentHandler: appointmentHandler}
}

func (r *Router) RegisterRoutes(g *gin.Engine) {
	g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := g.Group("/api/v1")
	{
		v1.POST("/appointments", r.appointmentHandler.BookAppointment)
		v1.GET("/appointments/available-slots", r.appointmentHandler.GetAvailableSlots)
	}
}
