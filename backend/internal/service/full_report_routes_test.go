package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/model"
)

type fakeFullReportRouteStore struct {
	rows []model.LLMModelConfig
}

func (s fakeFullReportRouteStore) ListEnabledRoutes(context.Context) ([]model.LLMModelConfig, error) {
	return s.rows, nil
}

func TestDatabaseFullReportRouteResolverFreezesOrderedNonSecretRoutes(t *testing.T) {
	cipher := llm.NewConfigSecretCipher("route-test-secret")
	keyCiphertext, err := cipher.Encrypt("private-test-key")
	if err != nil {
		t.Fatal(err)
	}
	store := fakeFullReportRouteStore{rows: []model.LLMModelConfig{{
		ID: 9, ProviderID: 7, Name: "自动模型", ModelID: "auto", Priority: 1, MaxRetries: 3, Enabled: true,
		Provider: model.LLMProviderConfig{ID: 7, Code: "custom", Name: "自定义", BaseURL: "http://host.docker.internal:18080/v1", APIKeyCiphertext: keyCiphertext, Enabled: true},
	}}}
	resolver := NewDatabaseFullReportRouteResolver(store, cipher, 45*time.Second)
	routes, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Provider == nil {
		t.Fatalf("unexpected resolved routes: %+v", routes)
	}
	frozen := routes[0].Frozen
	if frozen.Model != "auto" || frozen.MaxRetries != 3 || frozen.MaxAttempts != 4 || frozen.TimeoutSeconds != 45 || frozen.RouteNo != 1 {
		t.Fatalf("unexpected frozen route: %+v", frozen)
	}
	raw, err := json.Marshal(frozen)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "private-test-key") || strings.Contains(text, keyCiphertext) || strings.Contains(text, "api_key") {
		t.Fatalf("frozen route leaks credential material: %s", text)
	}
}
