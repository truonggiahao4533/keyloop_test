package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"keyloop-test/internal/domain"

	"github.com/redis/go-redis/v9"
)

const serviceKeyPrefix = "service:"

// ServiceCache implements usecase.ServiceCacheProvider using Redis.
type ServiceCache struct {
	client *redis.Client
}

func NewServiceCache(client *redis.Client) *ServiceCache {
	return &ServiceCache{client: client}
}

func (c *ServiceCache) GetService(ctx context.Context, id string) (*domain.Service, bool, error) {
	val, err := c.client.Get(ctx, serviceKeyPrefix+id).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var svc domain.Service
	if err := json.Unmarshal([]byte(val), &svc); err != nil {
		return nil, false, err
	}
	return &svc, true, nil
}

func (c *ServiceCache) SetService(ctx context.Context, id string, service *domain.Service, ttl time.Duration) error {
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, serviceKeyPrefix+id, data, ttl).Err()
}
