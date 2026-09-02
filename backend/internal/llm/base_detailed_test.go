package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleDetailedUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"{\"ok\":true}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`))
	}))
	defer server.Close()

	provider := newOpenAICompat("test", "secret", server.URL+"/v1", "auto")
	result, err := provider.GenerateJSONDetailed(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage.PromptTokens == nil || *result.Usage.PromptTokens != 11 || result.Usage.TotalTokens == nil || *result.Usage.TotalTokens != 18 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
}

func TestOpenAICompatibleMissingUsageRemainsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"{\"ok\":true}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	result, err := newOpenAICompat("test", "secret", server.URL+"/v1", "auto").GenerateJSONDetailed(context.Background(), "system", "user")
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage.PromptTokens != nil || result.Usage.CompletionTokens != nil || result.Usage.TotalTokens != nil {
		t.Fatalf("missing usage must remain nil: %+v", result.Usage)
	}
}
