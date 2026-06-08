package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"fatelumen/backend/internal/pkg/response"
)

// RateLimiter 简易内存限流器（按 IP token bucket，MVP 用内存实现）。
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	limit   int
	window  time.Duration
}

type tokenBucket struct {
	tokens   int
	resetAt  time.Time
}

func NewRateLimiter(limit int, windowSeconds int64) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		limit:   limit,
		window:  time.Duration(windowSeconds) * time.Second,
	}
}

func (rl *RateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		b, ok := rl.buckets[ip]
		if !ok || now.After(b.resetAt) {
			rl.buckets[ip] = &tokenBucket{tokens: 1, resetAt: now.Add(rl.window)}
			rl.mu.Unlock()
			c.Next()
			return
		}
		if b.tokens >= rl.limit {
			rl.mu.Unlock()
			response.Fail(c, response.CodeQuotaExhausted, "rate limit exceeded")
			c.Abort()
			return
		}
		b.tokens++
		rl.mu.Unlock()
		c.Next()
	}
}
