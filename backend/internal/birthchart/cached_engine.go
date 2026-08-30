package birthchart

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"fatelumen/backend/internal/cache"
	"fatelumen/backend/internal/pkg/logger"
)

type CachedEngine struct {
	next  Engine
	cache cache.Cache
	ttl   time.Duration
}

const cacheSchemaVersion = "birthchart-cache-v2"

func NewCachedEngine(next Engine, c cache.Cache, ttl time.Duration) Engine {
	if next == nil {
		next = NewDefaultEngine()
	}
	if c == nil {
		return next
	}
	return &CachedEngine{next: next, cache: c, ttl: ttl}
}

func (e *CachedEngine) Calculate(ctx context.Context, input Input) (*Result, error) {
	raw, _ := json.Marshal(struct {
		CacheSchema string `json:"cache_schema"`
		Version     string `json:"version"`
		Input       Input  `json:"input"`
	}{cacheSchemaVersion, EngineVersion, input})
	sum := sha256.Sum256(raw)
	key := fmt.Sprintf("birthchart:%x", sum)
	if value, err := e.cache.Get(ctx, key); err != nil {
		logger.FromCtx(ctx).Error("birth chart cache get failed", "err", err, "cache_key", key)
	} else if value != "" {
		var result Result
		if json.Unmarshal([]byte(value), &result) == nil {
			return &result, nil
		}
	}
	result, err := e.next.Calculate(ctx, input)
	if err != nil {
		return nil, err
	}
	if value, marshalErr := json.Marshal(result); marshalErr == nil {
		if err := e.cache.Set(ctx, key, string(value), e.ttl); err != nil {
			logger.FromCtx(ctx).Error("birth chart cache set failed", "err", err, "cache_key", key)
		}
	}
	return result, nil
}
