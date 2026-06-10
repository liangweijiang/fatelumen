package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_IncrFresh(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	count, err := mc.Incr(ctx, "test:key1")
	if err != nil {
		t.Fatalf("Incr failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
}

func TestMemoryCache_IncrAccumulate(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	key := "test:accumulate"
	for i := int64(1); i <= 5; i++ {
		count, err := mc.Incr(ctx, key)
		if err != nil {
			t.Fatalf("Incr #%d failed: %v", i, err)
		}
		if count != i {
			t.Fatalf("Incr #%d: expected %d, got %d", i, i, count)
		}
	}
}

func TestMemoryCache_GetSet(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	err := mc.Set(ctx, "test:greeting", "hello", 10*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := mc.Get(ctx, "test:greeting")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got '%s'", val)
	}

	val, err = mc.Get(ctx, "test:nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string, got '%s'", val)
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	err := mc.Set(ctx, "test:short", "alive", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := mc.Get(ctx, "test:short")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "alive" {
		t.Fatalf("expected 'alive', got '%s'", val)
	}

	time.Sleep(100 * time.Millisecond)

	val, err = mc.Get(ctx, "test:short")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty after expiry, got '%s'", val)
	}
}

func TestMemoryCache_IncrAfterExpiryResets(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	key := "test:expire_reset"

	count, _ := mc.Incr(ctx, key)
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}

	err := mc.Set(ctx, key, "1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	count, err = mc.Incr(ctx, key)
	if err != nil {
		t.Fatalf("Incr after expiry failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 after expiry reset, got %d", count)
	}
}

func TestMemoryCache_ConcurrentIncr(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	key := "test:concurrent"
	var wg sync.WaitGroup
	n := 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mc.Incr(ctx, key)
			if err != nil {
				t.Errorf("Incr failed: %v", err)
			}
		}()
	}
	wg.Wait()

	val, err := mc.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "100" {
		t.Fatalf("expected '100' after 100 concurrent incrs, got '%s'", val)
	}
}

func TestMemoryCache_SetOverwrite(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Close()
	ctx := context.Background()

	mc.Set(ctx, "test:ow", "first", time.Hour)
	mc.Set(ctx, "test:ow", "second", time.Hour)

	val, _ := mc.Get(ctx, "test:ow")
	if val != "second" {
		t.Fatalf("expected 'second', got '%s'", val)
	}
}
