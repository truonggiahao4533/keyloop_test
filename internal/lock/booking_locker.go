package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const bookingLockTTL = 30 * time.Second

// BookingLockerInterface allows the locker to be swapped out in tests.
type BookingLockerInterface interface {
	AcquireLock(ctx context.Context, technicianID, bayID string, start, end time.Time) (bool, error)
	ReleaseLock(ctx context.Context, technicianID, bayID string, start, end time.Time) error
}

type BookingLocker struct {
	client *redis.Client
}

func NewBookingLocker(client *redis.Client) *BookingLocker {
	return &BookingLocker{client: client}
}

// bookingLockKey builds the composite key that uniquely identifies a
// (technician, bay, timeslot) triple.
// Unix seconds in UTC are used so the key is compact and timezone-agnostic.
// Two bookings for the same technician+bay at DIFFERENT slots get different
// keys and never block each other.
func bookingLockKey(technicianID, bayID string, start, end time.Time) string {
	return fmt.Sprintf("lock:booking:tech:%s:bay:%s:slot:%d-%d",
		technicianID, bayID, start.UTC().Unix(), end.UTC().Unix())
}

// AcquireLock attempts a single atomic SetNX on the composite key with a
// 30-second TTL.
// Returns (true, nil) if the lock was acquired, (false, nil) if another
// request already holds it, or (false, err) if Redis is unreachable.
func (l *BookingLocker) AcquireLock(ctx context.Context, technicianID, bayID string, start, end time.Time) (bool, error) {
	key := bookingLockKey(technicianID, bayID, start, end)
	acquired, err := l.client.SetNX(ctx, key, 1, bookingLockTTL).Result()
	if err != nil {
		return false, err
	}
	return acquired, nil
}

// ReleaseLock deletes the composite key. If the key no longer exists
// (expired or was never set) the call is a no-op and returns nil.
func (l *BookingLocker) ReleaseLock(ctx context.Context, technicianID, bayID string, start, end time.Time) error {
	key := bookingLockKey(technicianID, bayID, start, end)
	return l.client.Del(ctx, key).Err()
}
