package birthchart

import (
	"context"
	"testing"
	"time"

	"fatelumen/backend/internal/cache"
)

type countingEngine struct{ calls int }

func (e *countingEngine) Calculate(_ context.Context, _ Input) (*Result, error) {
	e.calls++
	return &Result{EngineVersion: string(rune('0' + e.calls))}, nil
}

func TestCachedEngineCachesIdenticalPublicCalculation(t *testing.T) {
	memory := cache.NewMemoryCache()
	defer memory.Close()
	base := &countingEngine{}
	engine := NewCachedEngine(base, memory, time.Hour)

	first, err := engine.Calculate(context.Background(), Input{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.Calculate(context.Background(), Input{})
	if err != nil {
		t.Fatal(err)
	}
	if base.calls != 1 || first.EngineVersion != second.EngineVersion {
		t.Fatalf("normal request should use cache: calls=%d first=%q second=%q", base.calls, first.EngineVersion, second.EngineVersion)
	}
}
