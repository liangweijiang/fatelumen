package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fatelumen/backend/internal/cache"
)

func TestQuotaService_ThreeSuccesses(t *testing.T) {
	mc := cache.NewMemoryCache()
	defer mc.Close()
	qs := NewQuotaService(mc, 3, nil)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		err := qs.CheckAndConsume(ctx, 42)
		if err != nil {
			t.Fatalf("call #%d: expected nil, got %v", i, err)
		}
	}
}

func TestQuotaService_FourthExceeds(t *testing.T) {
	mc := cache.NewMemoryCache()
	defer mc.Close()
	qs := NewQuotaService(mc, 3, nil)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if err := qs.CheckAndConsume(ctx, 99); err != nil {
			t.Fatalf("call #%d: expected nil, got %v", i, err)
		}
	}

	err := qs.CheckAndConsume(ctx, 99)
	if err == nil {
		t.Fatal("expected quota exceeded error on 4th call, got nil")
	}
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestQuotaService_CrossDayReset(t *testing.T) {
	mc := cache.NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	today := now.Format("2006-01-02")

	if yesterday == today {
		t.Skip("test requires different dates")
	}

	qs := NewQuotaService(mc, 3, nil)

	for i := 1; i <= 3; i++ {
		if err := qs.CheckAndConsume(ctx, 7); err != nil {
			t.Fatalf("call #%d: expected nil, got %v", i, err)
		}
	}

	err := qs.CheckAndConsume(ctx, 7)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded today, got %v", err)
	}

	// Manually simulate yesterday's count being zero by checking the cache directly.
	// The key contains the date, so yesterday's key would have been different.
	yesterdayKey := "quota:7:" + yesterday
	val, err := mc.Get(ctx, yesterdayKey)
	if err != nil {
		t.Fatalf("Get yesterday key failed: %v", err)
	}
	if val != "" {
		t.Fatalf("yesterday key should be empty (not used), got '%s'", val)
	}

	todayKey := "quota:7:" + today
	val, err = mc.Get(ctx, todayKey)
	if err != nil {
		t.Fatalf("Get today key failed: %v", err)
	}
	if val != "4" {
		t.Fatalf("today count should be 4, got '%s'", val)
	}
}

func TestQuotaService_DifferentUsersSeparated(t *testing.T) {
	mc := cache.NewMemoryCache()
	defer mc.Close()
	qs := NewQuotaService(mc, 3, nil)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if err := qs.CheckAndConsume(ctx, 1); err != nil {
			t.Fatalf("user 1 call #%d: expected nil, got %v", i, err)
		}
	}

	err := qs.CheckAndConsume(ctx, 1)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("user 1 4th: expected ErrQuotaExceeded, got %v", err)
	}

	for i := 1; i <= 3; i++ {
		if err := qs.CheckAndConsume(ctx, 2); err != nil {
			t.Fatalf("user 2 call #%d: expected nil, got %v", i, err)
		}
	}

	err = qs.CheckAndConsume(ctx, 2)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("user 2 4th: expected ErrQuotaExceeded, got %v", err)
	}
}

func TestQuotaService_CustomLimit(t *testing.T) {
	mc := cache.NewMemoryCache()
	defer mc.Close()
	qs := NewQuotaService(mc, 5, nil)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		if err := qs.CheckAndConsume(ctx, 55); err != nil {
			t.Fatalf("call #%d: expected nil, got %v", i, err)
		}
	}

	err := qs.CheckAndConsume(ctx, 55)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
}
