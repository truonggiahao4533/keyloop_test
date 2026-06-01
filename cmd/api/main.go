// @title           Keyloop Appointment Booking API
// @version         1.0
// @description     RESTful API for booking vehicle service appointments at dealerships.
// @host            localhost:8081
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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load() // Load environment variables from .env file
	// Initialize Database
	postgre_host := os.Getenv("POSTGRES_HOST")
	postgre_port := os.Getenv("POSTGRES_PORT")
	postgre_user := os.Getenv("POSTGRES_USER")
	postgre_password := os.Getenv("POSTGRES_PASSWORD")
	postgre_dbName := os.Getenv("POSTGRES_DB")
	if postgre_host == "" {
		postgre_host = "localhost"
	}
	if postgre_port == "" {
		postgre_port = "5432"
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgre_user, postgre_password, postgre_host, postgre_port, postgre_dbName)

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
	serviceRepo := postgrerepository.NewServiceRepository(db)
	vehicleRepo := postgrerepository.NewVehicleRepository(db)
	availabilitySlotRepo := postgrerepository.NewAvailabilitySlotRepository(db)
	appointmentRepo := postgrerepository.NewAppointmentRepository(db)

	// Initialize Use Cases
	appointmentUC := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		serviceBayRepo,
		technicianRepo,
		serviceRepo,
		vehicleRepo,
		serviceCache,
		availabilitySlotRepo,
		appointmentRepo,
	)

	//Run gin server
	r := gin.Default()

	// Initialize Handlers and Routes
	appointmentHandler := httpHandler.NewAppointmentHandler(appointmentUC)
	router := httpHandler.NewRouter(appointmentHandler)
	router.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		panic("failed to start server: " + err.Error())
	}
}
