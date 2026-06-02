package rediscache

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a Redis client from REDIS_ADDR env var (default: localhost:6379).
func NewRedisClient() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		slog.Warn("redis ping failed, cache unavailable", "error", err)
	}
	return client
}
