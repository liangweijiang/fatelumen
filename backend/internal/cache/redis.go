package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache 基于 go-redis 的缓存实现。
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建 Redis 缓存实例。
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Incr 原子递增计数器。
func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Get 获取 key 对应的值。
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Set 写入 key-val 并设置 TTL。
func (r *RedisCache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return r.client.Set(ctx, key, val, ttl).Err()
}
