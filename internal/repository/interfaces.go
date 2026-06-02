package repository

import (
	"context"
	"time"

	"keyloop-test/internal/domain"
)

type CustomerRepository interface {
	GetCustomer(ctx context.Context, id string) (*domain.Customer, error)
	ListCustomers(ctx context.Context) ([]*domain.Customer, error)
	CreateCustomer(ctx context.Context, customer *domain.Customer) error
	UpdateCustomer(ctx context.Context, customer *domain.Customer) error
	DeleteCustomer(ctx context.Context, id string) error
}

type VehicleRepository interface {
	GetVehicle(ctx context.Context, id string) (*domain.Vehicle, error)
	ListVehiclesByCustomer(ctx context.Context, customerID string) ([]domain.Vehicle, error)
	CreateVehicle(ctx context.Context, vehicle *domain.Vehicle) error
	UpdateVehicle(ctx context.Context, vehicle *domain.Vehicle) error
	DeleteVehicle(ctx context.Context, id string) error
}

type DealershipRepository interface {
	GetDealership(ctx context.Context, id string) (*domain.Dealership, error)
	ListDealerships(ctx context.Context) ([]*domain.Dealership, error)
	CreateDealership(ctx context.Context, dealership *domain.Dealership) error
	UpdateDealership(ctx context.Context, dealership *domain.Dealership) error
	DeleteDealership(ctx context.Context, id string) error
}

type ServiceBayRepository interface {
	GetServiceBay(ctx context.Context, id string) (*domain.ServiceBay, error)
	ListServiceBaysByDealership(ctx context.Context, dealershipID string) ([]*domain.ServiceBay, error)
	CreateServiceBay(ctx context.Context, bay *domain.ServiceBay) error
	UpdateServiceBay(ctx context.Context, bay *domain.ServiceBay) error
	DeleteServiceBay(ctx context.Context, id string) error
}

type ServiceRepository interface {
	GetService(ctx context.Context, id string) (*domain.Service, error)
	ListServices(ctx context.Context) ([]*domain.Service, error)
	CreateService(ctx context.Context, service *domain.Service) error
	UpdateService(ctx context.Context, service *domain.Service) error
	DeleteService(ctx context.Context, id string) error
}

type TechnicianRepository interface {
	GetTechnician(ctx context.Context, id string) (*domain.Technician, error)
	ListTechniciansByDealership(ctx context.Context, dealershipID string) ([]*domain.Technician, error)
	CreateTechnician(ctx context.Context, tech *domain.Technician) error
	UpdateTechnician(ctx context.Context, tech *domain.Technician) error
	DeleteTechnician(ctx context.Context, id string) error
}

type TechnicianSkillRepository interface {
	ListTechnicianSkills(ctx context.Context, technicianID string) ([]*domain.TechnicianSkill, error)
	CreateTechnicianSkill(ctx context.Context, skill *domain.TechnicianSkill) error
	DeleteTechnicianSkill(ctx context.Context, technicianID string, skill domain.ServiceType) error
}

type AppointmentRepository interface {
	GetAppointment(ctx context.Context, id string) (*domain.Appointment, error)
	ListAppointmentsByDealership(ctx context.Context, dealershipID string) ([]domain.Appointment, error)
	ListAppointmentsByCustomer(ctx context.Context, customerID string) ([]domain.Appointment, error)
	ListAppointmentsByCustomerAndDealership(ctx context.Context, customerID, dealershipID string) ([]domain.Appointment, error)
	CreateAppointment(ctx context.Context, appt *domain.Appointment) (*domain.Appointment, error)
	UpdateAppointmentStatus(ctx context.Context, appt *domain.Appointment, status domain.AppointmentStatus) error
	UpdateAppointment(ctx context.Context, id string, status domain.AppointmentStatus, notes string) (*domain.Appointment, error)
	DeleteAppointment(ctx context.Context, id string) error
}

type AvailabilitySlotRepository interface {
	GetAvailableSlots(ctx context.Context, startTime, endTime time.Time, dealershipID string, duration time.Duration, serviceTypes []string) ([]domain.AvailableSlot, error)
}
