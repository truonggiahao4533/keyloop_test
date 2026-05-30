package domain_test

import "keyloop-test/internal/domain"

import "testing"

func TestVehicle_DisplayName(t *testing.T) {
	veh := &domain.Vehicle{
		Make:  "Toyota",
		Model: "Camry",
	}

	expected := "Toyota Camry"
	if got := veh.DisplayName(); got != expected {
		t.Errorf("DisplayName() = %v, want %v", got, expected)
	}
}
