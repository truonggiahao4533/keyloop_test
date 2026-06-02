package domain_test

import (
	"testing"
	"time"

	"keyloop-test/internal/domain"
)

// dealership open Mon–Fri 08:00–18:00
func weekdayDealershipD() *domain.Dealership {
	return &domain.Dealership{
		OpenTime:    8 * time.Hour,
		CloseTime:   18 * time.Hour,
		WorkingDays: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
	}
}

// A known Monday in UTC used as a stable anchor for weekday arithmetic.
var anchorMonday = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // 2026-06-01 is a Monday

func TestDealership_IsWorkingDay(t *testing.T) {
	d := weekdayDealershipD()
	tests := []struct {
		name string
		day  time.Time
		want bool
	}{
		{"monday is working", anchorMonday, true},
		{"tuesday is working", anchorMonday.Add(24 * time.Hour), true},
		{"wednesday is working", anchorMonday.Add(48 * time.Hour), true},
		{"thursday is working", anchorMonday.Add(72 * time.Hour), true},
		{"friday is working", anchorMonday.Add(96 * time.Hour), true},
		{"saturday is not working", anchorMonday.Add(120 * time.Hour), false},
		{"sunday is not working", anchorMonday.Add(144 * time.Hour), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.IsWorkingDay(tt.day); got != tt.want {
				t.Errorf("IsWorkingDay(%v) = %v, want %v", tt.day.Weekday(), got, tt.want)
			}
		})
	}
}

func TestDealership_IsWorkingDay_EmptyWorkingDays(t *testing.T) {
	d := &domain.Dealership{WorkingDays: []time.Weekday{}}
	if d.IsWorkingDay(anchorMonday) {
		t.Error("IsWorkingDay should be false when WorkingDays is empty")
	}
}

func TestDealership_IsSlotWithinHours(t *testing.T) {
	d := weekdayDealershipD()
	base := anchorMonday // 2026-06-01, Monday midnight UTC

	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  bool
	}{
		{
			name:  "slot within hours",
			start: base.Add(9 * time.Hour),  // 09:00
			end:   base.Add(10 * time.Hour), // 10:00
			want:  true,
		},
		{
			name:  "starts at open, ends before close",
			start: base.Add(8 * time.Hour),              // 08:00 open
			end:   base.Add(8*time.Hour + 30*time.Minute), // 08:30
			want:  true,
		},
		{
			name:  "starts before open",
			start: base.Add(7 * time.Hour + 30*time.Minute), // 07:30
			end:   base.Add(9 * time.Hour),
			want:  false,
		},
		{
			name:  "ends after close",
			start: base.Add(17*time.Hour + 30*time.Minute), // 17:30
			end:   base.Add(18*time.Hour + 30*time.Minute), // 18:30
			want:  false,
		},
		{
			name:  "ends exactly at close",
			start: base.Add(17*time.Hour + 30*time.Minute), // 17:30
			end:   base.Add(18 * time.Hour),                // 18:00 exactly
			want:  true,
		},
		{
			name:  "starts exactly at open",
			start: base.Add(8 * time.Hour),  // 08:00 exactly
			end:   base.Add(9 * time.Hour),  // 09:00
			want:  true,
		},
		{
			name:  "full day slot open to close",
			start: base.Add(8 * time.Hour),  // 08:00
			end:   base.Add(18 * time.Hour), // 18:00
			want:  true,
		},
		{
			name: "cross-midnight slot rejected",
			// midnight (00:00) of the next day — timeOfDay == 0 <= CloseTime is always true
			// without the date guard, this would incorrectly pass
			start: base.Add(23 * time.Hour),                 // 23:00 day 1
			end:   base.Add(24*time.Hour + 1*time.Hour),     // 01:00 day 2
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.IsSlotWithinHours(tt.start, tt.end); got != tt.want {
				t.Errorf("IsSlotWithinHours(%v, %v) = %v, want %v",
					tt.start.Format(time.TimeOnly), tt.end.Format(time.TimeOnly), got, tt.want)
			}
		})
	}
}
