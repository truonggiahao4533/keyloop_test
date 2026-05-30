package domain_test

import "keyloop-test/internal/domain"

import (
	"testing"
	"time"
)

func TestAppointment_Duration(t *testing.T) {
	start := time.Date(2023, 10, 25, 9, 0, 0, 0, time.UTC)
	end := time.Date(2023, 10, 25, 10, 30, 0, 0, time.UTC)

	appt := &domain.Appointment{
		StartTime: start,
		EndTime:   end,
	}

	expected := 90 * time.Minute
	if got := appt.Duration(); got != expected {
		t.Errorf("Duration() = %v, want %v", got, expected)
	}
}

func TestAppointment_IsConfirmed(t *testing.T) {
	tests := []struct {
		name   string
		status domain.AppointmentStatus
		want   bool
	}{
		{"confirmed", domain.AppointmentStatusConfirmed, true},
		{"pending", domain.AppointmentStatusPending, false},
		{"cancelled", domain.AppointmentStatusCancelled, false},
		{"completed", domain.AppointmentStatusCompleted, false},
		{"no show", domain.AppointmentStatusNoShow, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appt := &domain.Appointment{Status: tt.status}
			if got := appt.IsConfirmed(); got != tt.want {
				t.Errorf("IsConfirmed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppointment_CanCancel(t *testing.T) {
	tests := []struct {
		name   string
		status domain.AppointmentStatus
		want   bool
	}{
		{"pending", domain.AppointmentStatusPending, true},
		{"confirmed", domain.AppointmentStatusConfirmed, true},
		{"cancelled", domain.AppointmentStatusCancelled, false},
		{"completed", domain.AppointmentStatusCompleted, false},
		{"no show", domain.AppointmentStatusNoShow, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appt := &domain.Appointment{Status: tt.status}
			if got := appt.CanCancel(); got != tt.want {
				t.Errorf("CanCancel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppointment_OverlapsWith(t *testing.T) {
	baseStart := time.Date(2023, 10, 25, 10, 0, 0, 0, time.UTC)
	baseEnd := time.Date(2023, 10, 25, 12, 0, 0, 0, time.UTC)

	appt := &domain.Appointment{
		StartTime: baseStart,
		EndTime:   baseEnd,
	}

	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  bool
	}{
		{
			name:  "exact match",
			start: baseStart,
			end:   baseEnd,
			want:  true,
		},
		{
			name:  "completely inside",
			start: baseStart.Add(30 * time.Minute),
			end:   baseEnd.Add(-30 * time.Minute),
			want:  true,
		},
		{
			name:  "completely outside",
			start: baseStart.Add(-1 * time.Hour),
			end:   baseEnd.Add(1 * time.Hour),
			want:  true,
		},
		{
			name:  "overlap start",
			start: baseStart.Add(-1 * time.Hour),
			end:   baseStart.Add(1 * time.Hour),
			want:  true,
		},
		{
			name:  "overlap end",
			start: baseEnd.Add(-1 * time.Hour),
			end:   baseEnd.Add(1 * time.Hour),
			want:  true,
		},
		{
			name:  "before, no overlap",
			start: baseStart.Add(-2 * time.Hour),
			end:   baseStart.Add(-1 * time.Hour),
			want:  false,
		},
		{
			name:  "after, no overlap",
			start: baseEnd.Add(1 * time.Hour),
			end:   baseEnd.Add(2 * time.Hour),
			want:  false,
		},
		{
			name:  "touching start, no overlap",
			start: baseStart.Add(-1 * time.Hour),
			end:   baseStart,
			want:  false,
		},
		{
			name:  "touching end, no overlap",
			start: baseEnd,
			end:   baseEnd.Add(1 * time.Hour),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appt.OverlapsWith(tt.start, tt.end); got != tt.want {
				t.Errorf("OverlapsWith() = %v, want %v", got, tt.want)
			}
		})
	}
}
