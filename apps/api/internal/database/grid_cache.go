package database

import (
	"context"
	"time"

	"github.com/ecollm/api/internal/carbon"
	"github.com/redis/go-redis/v9"
)

type redisGridCache struct {
	client *redis.Client
}

// NewGridCacheAdapter wraps a Redis client to satisfy carbon.GridCache.
func NewGridCacheAdapter(client *redis.Client) carbon.GridCache {
	return &redisGridCache{client: client}
}

func (r *redisGridCache) Get(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

func (r *redisGridCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}
