package llm

import (
	"context"
	"errors"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestClassifyCallError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{"timeout", context.DeadlineExceeded, "LLM_TIMEOUT"},
		{"rate limit", &openai.APIError{HTTPStatusCode: 429}, "LLM_RATE_LIMITED"},
		{"authentication", &openai.RequestError{HTTPStatusCode: 401}, "LLM_AUTHENTICATION"},
		{"upstream", &openai.APIError{HTTPStatusCode: 503}, "LLM_UPSTREAM"},
		{"unknown", errors.New("opaque failure"), "LLM_PROVIDER_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyCallError(tt.err); got.Code != tt.code || got.Summary == "" {
				t.Fatalf("got %+v, want code %s", got, tt.code)
			}
		})
	}
}
