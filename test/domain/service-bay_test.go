package domain_test

import "keyloop-test/internal/domain"

import "testing"

func TestServiceBay_IsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		status domain.BayStatus
		want   bool
	}{
		{"active", domain.BayStatusActive, true},
		{"inactive", domain.BayStatusInactive, false},
		{"maintenance", domain.BayStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := &domain.ServiceBay{Status: tt.status}
			if got := sb.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
