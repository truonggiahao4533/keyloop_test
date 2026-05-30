package main

import (
	"database/sql"
	"fmt"
	"os"

	repository "keyloop-test/infrastructure/postgresql/postgre-repository"
	"keyloop-test/internal/usecase"

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

	// Initialize Repository
	dealershipRepo := repository.NewDealershipRepository(db)
	serviceBayRepo := repository.NewServiceBayRepository(db)
	technicianRepo := repository.NewTechnicianRepository(db)
	reservationRepo := repository.NewReservationRepository(db)

	// Initialize Use Case
	_ = usecase.NewAppointmentBookingUseCase(
		dealershipRepo,
		serviceBayRepo,
		technicianRepo,
		reservationRepo,
	)

	// TODO: Start HTTP Server
	// router := gin.Default()
	// api := NewAPI(router, appoinmentBookingUseCase)
	// api.SetupRoutes()
	// router.Run(":8080")

	fmt.Println("Server started successfully")
}
