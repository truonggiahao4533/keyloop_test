package usecase

import (
	"context"
	"time"

	"keyloop-test/internal/domain"
)

// ServiceCacheProvider defines caching operations for Service catalogue entries.
// Implementations live in infrastructure/redis; the bool return signals a cache hit.
type ServiceCacheProvider interface {
	GetService(ctx context.Context, id string) (*domain.Service, bool, error)
	SetService(ctx context.Context, id string, service *domain.Service, ttl time.Duration) error
}
