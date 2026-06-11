package ratelimit

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestMemoryLimiter_AllowsWithinLimit(t *testing.T) {
	lim := NewMemoryLimiter(rate.Limit(10), 10)
	key := "uid:1"

	for i := 0; i < 10; i++ {
		allowed, _ := lim.Allow(key)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestMemoryLimiter_RejectsBeyondLimit(t *testing.T) {
	lim := NewMemoryLimiter(rate.Limit(10), 5)
	key := "uid:1"

	allowed := 0
	for i := 0; i < 20; i++ {
		ok, _ := lim.Allow(key)
		if ok {
			allowed++
		}
	}
	if allowed > 10 {
		t.Errorf("allowed %d requests, expected at most 10 in burst filling window", allowed)
	}
	if allowed < 5 {
		t.Errorf("allowed only %d requests, expected at least burst of 5", allowed)
	}
}

func TestMemoryLimiter_RejectReturnsRetryAfter(t *testing.T) {
	lim := NewMemoryLimiter(rate.Limit(1), 1)
	key := "uid:1"

	// Consume the burst token
	lim.Allow(key)
	// Next call should be rejected
	allowed, retry := lim.Allow(key)
	if allowed {
		t.Fatal("second call should be rejected after burst exhausted")
	}
	if retry <= 0 {
		t.Errorf("retryAfter should be positive, got %v", retry)
	}
}

func TestMemoryLimiter_DifferentKeysIsolated(t *testing.T) {
	lim := NewMemoryLimiter(rate.Limit(1), 1)

	// Exhaust key A
	lim.Allow("uid:1")
	_, _ = lim.Allow("uid:1") // should reject

	// Key B should still be allowed
	allowed, _ := lim.Allow("uid:2")
	if !allowed {
		t.Fatal("different key should be isolated and allowed")
	}
}

func TestMemoryLimiter_RecoveryAfterWait(t *testing.T) {
	// Use a very permissive limiter for this timing test
	lim := NewMemoryLimiter(rate.Limit(100), 1)
	key := "uid:1"

	// Consume the single burst token
	lim.Allow(key)

	// Should be rejected immediately
	allowed, _ := lim.Allow(key)
	if allowed {
		t.Log("second request unexpectedly allowed (timing-dependent)")
	}

	// After a short wait, new token should be available
	time.Sleep(15 * time.Millisecond)
	allowed, _ = lim.Allow(key)
	if !allowed {
		t.Fatal("request should be allowed after waiting for token regeneration")
	}
}

func TestMemoryLimiter_ZeroBurst(t *testing.T) {
	lim := NewMemoryLimiter(rate.Limit(100), 0)
	key := "uid:1"

	allowed, _ := lim.Allow(key)
	if allowed {
		t.Log("zero burst with high rate may allow immediately")
	}
}
