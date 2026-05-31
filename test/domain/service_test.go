package domain_test

import "keyloop-test/internal/domain"

import (
	"testing"
	"time"
)

func TestService_EstimatedDuration(t *testing.T) {
	sd := &domain.Service{
		EstimatedMinutes: 45,
	}

	expected := 45 * time.Minute
	if got := sd.EstimatedDuration(); got != expected {
		t.Errorf("EstimatedDuration() = %v, want %v", got, expected)
	}
}
