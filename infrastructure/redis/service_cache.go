package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"keyloop-test/internal/domain"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const serviceKeyPrefix = "service:"

// ServiceCache implements usecase.ServiceCacheProvider using Redis.
type ServiceCache struct {
	client *redis.Client
	tracer trace.Tracer
}

func NewServiceCache(client *redis.Client) *ServiceCache {
	return &ServiceCache{
		client: client,
		tracer: otel.Tracer("keyloop-test/cache"),
	}
}

func (c *ServiceCache) GetService(ctx context.Context, id string) (*domain.Service, bool, error) {
	ctx, span := c.tracer.Start(ctx, "cache.GetService",
		trace.WithAttributes(attribute.String("cache.key", serviceKeyPrefix+id)),
	)
	defer span.End()

	val, err := c.client.Get(ctx, serviceKeyPrefix+id).Result()
	if errors.Is(err, redis.Nil) {
		span.SetAttributes(attribute.Bool("cache.hit", false))
		return nil, false, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, false, err
	}
	span.SetAttributes(attribute.Bool("cache.hit", true))
	var svc domain.Service
	if err := json.Unmarshal([]byte(val), &svc); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, false, err
	}
	return &svc, true, nil
}

func (c *ServiceCache) SetService(ctx context.Context, id string, service *domain.Service, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "cache.SetService",
		trace.WithAttributes(
			attribute.String("cache.key", serviceKeyPrefix+id),
			attribute.String("cache.ttl", ttl.String()),
		),
	)
	defer span.End()

	data, err := json.Marshal(service)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := c.client.Set(ctx, serviceKeyPrefix+id, data, ttl).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
