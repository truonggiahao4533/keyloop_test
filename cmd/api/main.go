// @title           Keyloop Appointment Booking API
// @version         1.0
// @description     RESTful API for booking vehicle service appointments at dealerships.
// @host            localhost:8080
// @BasePath        /

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "keyloop-test/docs"
	postgrerepository "keyloop-test/infrastructure/postgresql/postgre-repository"
	rediscache "keyloop-test/infrastructure/redis"
	httpHandler "keyloop-test/internal/adapter/http"
	"keyloop-test/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()

	// Initialize Redis
	redisClient := rediscache.NewRedisClient()
	defer redisClient.Close()
	serviceCache := rediscache.NewServiceCache(redisClient)

	// Initialize Repositories
	dealershipRepo := postgrerepository.NewDealershipRepository(db)
	serviceBayRepo := postgrerepository.NewServiceBayRepository(db)
	technicianRepo := postgrerepository.NewTechnicianRepository(db)
	serviceDefRepo := postgrerepository.NewServiceDefinitionRepository(db)
	vehicleRepo := postgrerepository.NewVehicleRepository(db)
	availabilitySlotRepo := postgrerepository.NewAvailabilitySlotRepository(db)
	appointmentRepo := postgrerepository.NewAppointmentRepository(db)

	// Initialize Use Cases
	appointmentUC := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		serviceBayRepo,
		technicianRepo,
		serviceDefRepo,
		vehicleRepo,
		serviceCache,
		availabilitySlotRepo,
		appointmentRepo,
	)

	//Run gin server
	router := gin.Default()
	httpHandler.NewAppointmentHandler(appointmentUC).RegisterRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server listening on :%s\n", port)
	if err := router.Run(":" + port); err != nil {
		panic("failed to start server: " + err.Error())
	}
}
