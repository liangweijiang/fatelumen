package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter is the rate-limiting interface. A Redis-backed implementation
// can be swapped in without changing middleware or business code.
type Limiter interface {
	// Allow reports whether the event for key may happen at this instant.
	// retryAfter is the suggested wait duration before the next attempt.
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

// memoryLimiter implements Limiter with a per-key token bucket backed by
// golang.org/x/time/rate. Suitable for single-instance deployments.
type memoryLimiter struct {
	mu     sync.Mutex
	limits map[string]*rate.Limiter
	r      rate.Limit
	burst  int
}

// NewMemoryLimiter creates an in-memory Limiter.
// r is the sustained rate (e.g. 5 tokens per second = rate.Limit(5)),
// burst is the maximum burst size.
func NewMemoryLimiter(r rate.Limit, burst int) Limiter {
	return &memoryLimiter{
		limits: make(map[string]*rate.Limiter),
		r:      r,
		burst:  burst,
	}
}

func (m *memoryLimiter) Allow(key string) (bool, time.Duration) {
	m.mu.Lock()
	lim, ok := m.limits[key]
	if !ok {
		lim = rate.NewLimiter(m.r, m.burst)
		m.limits[key] = lim
	}
	m.mu.Unlock()

	if lim.Allow() {
		return true, 0
	}

	// Estimate retryAfter: 1 / rate in seconds (ceiling to 1s minimum).
	retry := time.Duration(float64(time.Second) / float64(m.r))
	if retry < time.Second {
		retry = time.Second
	}
	return false, retry
}
