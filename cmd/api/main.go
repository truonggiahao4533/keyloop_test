// @title           Keyloop Appointment Booking API
// @version         1.0
// @description     RESTful API for booking vehicle service appointments at dealerships.
// @host            localhost:8081
// @BasePath        /

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "keyloop-test/docs"
	postgrerepository "keyloop-test/infrastructure/postgresql/postgre-repository"
	rediscache "keyloop-test/infrastructure/redis"
	"keyloop-test/infrastructure/telemetry"
	httpHandler "keyloop-test/infrastructure/http"
	"keyloop-test/internal/lock"
	"keyloop-test/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	godotenv.Load()

	otelShutdown, err := telemetry.SetupOTel(ctx)
	if err != nil {
		panic("failed to setup OTel: " + err.Error())
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			slog.Error("OTel shutdown failed", "error", err)
		}
	}()

	slog.SetDefault(slog.New(otelslog.NewHandler("keyloop-api")))

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
	bookingRepo := postgrerepository.NewUnifiedRepository(db)
	locker := lock.NewBookingLocker(redisClient)

	// Initialize Use Cases
	usecaseTracer := telemetry.NewTracer("keyloop-test/usecase")
	appointmentUC := usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		serviceBayRepo,
		technicianRepo,
		serviceRepo,
		vehicleRepo,
		serviceCache,
		availabilitySlotRepo,
		appointmentRepo,
		bookingRepo,
		locker,
		usecaseTracer,
	)

	// Warm up service cache from DB at startup
	if err := appointmentUC.WarmCache(ctx); err != nil {
		slog.Warn("cache warm-up failed", "error", err)
	}

	// Initialize Handlers and Routes
	r := gin.Default()
	appointmentHandler := httpHandler.NewAppointmentHandler(appointmentUC)
	router := httpHandler.NewRouter(appointmentHandler)
	router.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server listening", "port", port)
	if err := r.Run(":" + port); err != nil {
		slog.Error("server error", "error", err)
	}
}
