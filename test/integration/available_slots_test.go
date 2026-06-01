//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	postgrerepository "keyloop-test/infrastructure/postgresql/postgre-repository"
	"keyloop-test/internal/domain"
)

// bookAt creates an appointment via the real repo and fails the test on error.
func bookAt(t *testing.T, dealershipID string, workerIdx int, start time.Time, duration time.Duration) {
	t.Helper()
	repo := postgrerepository.NewAppointmentRepository(testDB)
	_, err := repo.CreateAppointment(context.Background(), &domain.Appointment{
		CustomerID:   fix.customers[workerIdx],
		VehicleID:    fix.vehicles[workerIdx],
		DealershipID: dealershipID,
		StartTime:    start,
		EndTime:      start.Add(duration),
		Services:     oilChangeServices(),
		Status:       domain.AppointmentStatusPending,
	})
	if err != nil {
		t.Fatalf("bookAt(%v, %v): %v", start, duration, err)
	}
}

// containsSlot returns true if any slot starts at the given hour:minute (UTC).
func containsSlot(slots []domain.AvailableSlot, hour, minute int) bool {
	for _, s := range slots {
		if s.Start.Hour() == hour && s.Start.Minute() == minute {
			return true
		}
	}
	return false
}

// searchWindow returns the start/end range covering a full business day tomorrow.
func searchWindow() (start, end time.Time) {
	return slotAt(8), slotAt(18)
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TestGetAvailableSlots_NoAppointments_ReturnsSlotsInRange confirms that the
// query returns slots when nothing is booked — basic sanity check.
func TestGetAvailableSlots_NoAppointments_ReturnsSlotsInRange(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership1, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected slots with no existing appointments, got none")
	}
	if !containsSlot(slots, 10, 0) {
		t.Error("expected 10:00 slot to be available with no bookings")
	}
}

// TestGetAvailableSlots_BookedSlot_Excluded verifies that a slot occupied by a
// pending appointment is absent from the results.
func TestGetAvailableSlots_BookedSlot_Excluded(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	bookAt(t, fix.dealership1, 0, slotAt(10), 30*time.Minute)

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership1, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if containsSlot(slots, 10, 0) {
		t.Error("10:00 should not be available after booking it")
	}
	// Unrelated slots must not be affected.
	if !containsSlot(slots, 11, 0) {
		t.Error("11:00 should still be available — no appointment there")
	}
}

// TestGetAvailableSlots_PartialOverlapSlots_Excluded checks that slots whose
// [start, start+duration) window overlaps a booking are excluded, while a slot
// whose window does NOT overlap remains available.
//
// Setup: appointment at 10:15–10:45.
//   - 10:00 slot → [10:00,10:30) overlaps [10:15,10:45) → excluded
//   - 10:30 slot → [10:30,11:00) overlaps [10:15,10:45) → excluded
//   - 09:30 slot → [09:30,10:00) does not overlap [10:15,10:45) → available
func TestGetAvailableSlots_PartialOverlapSlots_Excluded(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	bookAt(t, fix.dealership1, 0, slotAt(10).Add(15*time.Minute), 30*time.Minute) // 10:15–10:45

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership1, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if containsSlot(slots, 10, 0) {
		t.Error("10:00 should be excluded: [10:00,10:30) overlaps booked [10:15,10:45)")
	}
	if containsSlot(slots, 10, 30) {
		t.Error("10:30 should be excluded: [10:30,11:00) overlaps booked [10:15,10:45)")
	}
	if !containsSlot(slots, 9, 30) {
		t.Error("09:30 should be available: [09:30,10:00) does not overlap booked [10:15,10:45)")
	}
}

// TestGetAvailableSlots_AdjacentSlot_Available confirms that a slot starting
// exactly when a booking ends is available. The GIST range is [start, end)
// — upper-exclusive — so adjacent ranges do not overlap.
func TestGetAvailableSlots_AdjacentSlot_Available(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	bookAt(t, fix.dealership1, 0, slotAt(10), 30*time.Minute) // 10:00–10:30

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership1, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if containsSlot(slots, 10, 0) {
		t.Error("10:00 should not be available (booked)")
	}
	if !containsSlot(slots, 10, 30) {
		t.Error("10:30 should be available: [10:30,11:00) does not overlap [10:00,10:30)")
	}
}

// TestGetAvailableSlots_CancelledAppointment_SlotAvailable confirms that a
// cancelled appointment does not block its former time slot.
func TestGetAvailableSlots_CancelledAppointment_SlotAvailable(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	apptRepo := postgrerepository.NewAppointmentRepository(testDB)
	appt, err := apptRepo.CreateAppointment(context.Background(), &domain.Appointment{
		CustomerID:   fix.customers[0],
		VehicleID:    fix.vehicles[0],
		DealershipID: fix.dealership1,
		StartTime:    slotAt(10),
		EndTime:      slotAt(10).Add(30 * time.Minute),
		Services:     oilChangeServices(),
		Status:       domain.AppointmentStatusPending,
	})
	if err != nil {
		t.Fatalf("create appointment: %v", err)
	}
	if err := apptRepo.UpdateAppointmentStatus(context.Background(), appt, domain.AppointmentStatusCancelled); err != nil {
		t.Fatalf("cancel appointment: %v", err)
	}

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership1, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if !containsSlot(slots, 10, 0) {
		t.Error("10:00 should be available: the only appointment there was cancelled")
	}
}

// TestGetAvailableSlots_CapacityTwo_OneBooked_SlotStillAvailable checks that
// booking one of two available bays/technicians on dealership2 does not remove
// the slot from results — one resource is still free.
func TestGetAvailableSlots_CapacityTwo_OneBooked_SlotStillAvailable(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	bookAt(t, fix.dealership2, 0, slotAt(10), 30*time.Minute)

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership2, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if !containsSlot(slots, 10, 0) {
		t.Error("10:00 should still be available: only 1 of 2 bays/techs is booked")
	}
}

// TestGetAvailableSlots_CapacityTwo_BothBooked_SlotUnavailable verifies that
// filling all bays and technicians removes the slot, while other times remain.
func TestGetAvailableSlots_CapacityTwo_BothBooked_SlotUnavailable(t *testing.T) {
	t.Cleanup(func() { cleanAppointments(t) })

	bookAt(t, fix.dealership2, 0, slotAt(10), 30*time.Minute)
	bookAt(t, fix.dealership2, 1, slotAt(10), 30*time.Minute)

	repo := postgrerepository.NewAvailabilitySlotRepository(testDB)
	start, end := searchWindow()
	slots, err := repo.GetAvailableSlots(context.Background(), start, end, fix.dealership2, 30*time.Minute, []string{fix.serviceID})
	if err != nil {
		t.Fatalf("GetAvailableSlots: %v", err)
	}
	if containsSlot(slots, 10, 0) {
		t.Error("10:00 should not be available: both bays and technicians are booked")
	}
	// Other slots must remain unaffected.
	if !containsSlot(slots, 11, 0) {
		t.Error("11:00 should still be available — no appointments there")
	}
}
