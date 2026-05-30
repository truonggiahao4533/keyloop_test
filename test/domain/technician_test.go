package domain_test

import "keyloop-test/internal/domain"

import "testing"

func TestTechnician_FullName(t *testing.T) {
	tech := &domain.Technician{
		FirstName: "John",
		LastName:  "Doe",
	}

	expected := "John Doe"
	if got := tech.FullName(); got != expected {
		t.Errorf("FullName() = %v, want %v", got, expected)
	}
}

func TestTechnician_IsQualifiedFor(t *testing.T) {
	tech := &domain.Technician{
		Skills: []domain.ServiceType{domain.ServiceOilChange, domain.ServiceTireRotation},
	}

	tests := []struct {
		name        string
		serviceType domain.ServiceType
		want        bool
	}{
		{"qualified for oil change", domain.ServiceOilChange, true},
		{"qualified for tire rotation", domain.ServiceTireRotation, true},
		{"not qualified for full service", domain.ServiceFullService, false},
		{"not qualified for brake inspection", domain.ServiceBrakeInspection, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tech.IsQualifiedFor(tt.serviceType); got != tt.want {
				t.Errorf("IsQualifiedFor(%v) = %v, want %v", tt.serviceType, got, tt.want)
			}
		})
	}
}

func TestTechnician_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status domain.TechnicianStatus
		want   bool
	}{
		{"active", domain.TechnicianStatusActive, true},
		{"inactive", domain.TechnicianStatusInactive, false},
		{"on leave", domain.TechnicianStatusOnLeave, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tech := &domain.Technician{Status: tt.status}
			if got := tech.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
