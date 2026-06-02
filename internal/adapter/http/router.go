package httpHandler

import (
	_ "keyloop-test/docs"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Router struct {
	appointmentHandler *AppointmentHandler
}

func NewRouter(appointmentHandler *AppointmentHandler) *Router {
	return &Router{appointmentHandler: appointmentHandler}
}

func (r *Router) RegisterRoutes(g *gin.Engine) {
	g.Use(otelgin.Middleware("keyloop-api"))

	g.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := g.Group("/api/v1")
	{
		v1.POST("/appointments", r.appointmentHandler.BookAppointment)
		v1.GET("/appointments", r.appointmentHandler.ListAppointments)
		v1.GET("/appointments/available-slots", r.appointmentHandler.GetAvailableSlots)
	}
}
