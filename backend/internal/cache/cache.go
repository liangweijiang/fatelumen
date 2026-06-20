package cache

import (
	"context"
	"time"
)

// Cache 抽象 KV 缓存（Redis / 内存兜底）。
type Cache interface {
	Incr(ctx context.Context, key string) (int64, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, val string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}
