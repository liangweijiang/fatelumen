package llm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"fatelumen/backend/internal/pkg/logger"

	openai "github.com/sashabaranov/go-openai"
)

type openAICompatProvider struct {
	name   string
	client *openai.Client
	model  string
}

func newOpenAICompat(name, apiKey, baseURL, model string) *openAICompatProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return &openAICompatProvider{
		name:   name,
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// NewOpenAICompatibleProvider builds a provider for an administrator-managed
// OpenAI-compatible endpoint. The credential only lives in this client.
func NewOpenAICompatibleProvider(name, apiKey, baseURL, model string) LLMProvider {
	return newOpenAICompat(name, apiKey, baseURL, model)
}

func (p *openAICompatProvider) Name() string {
	return p.name
}

func (p *openAICompatProvider) GenerateJSON(ctx context.Context, system, user string, opts ...Option) (string, error) {
	result, err := p.GenerateJSONDetailed(ctx, system, user, opts...)
	return result.Content, err
}

func (p *openAICompatProvider) GenerateJSONDetailed(ctx context.Context, system, user string, opts ...Option) (GenerationResult, error) {
	cc := &callConfig{temperature: 0.7, maxTokens: 600}
	for _, o := range opts {
		o(cc)
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: system},
		{Role: openai.ChatMessageRoleUser, Content: user},
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:          p.model,
		Messages:       messages,
		Temperature:    float32(cc.temperature),
		MaxTokens:      cc.maxTokens,
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	})
	if err != nil {
		logger.FromCtx(ctx).Error("llm call failed", "err", err, "provider", p.name, "model", p.model, "elapsed_ms", time.Since(start).Milliseconds())
		return GenerationResult{}, err
	}
	logger.FromCtx(ctx).Info("llm call completed", "provider", p.name, "model", p.model, "elapsed_ms", time.Since(start).Milliseconds())

	if len(resp.Choices) == 0 {
		logger.FromCtx(ctx).Error("llm returned empty choices", "provider", p.name, "model", p.model)
		return GenerationResult{}, errors.New("llm returned empty response")
	}
	if resp.Choices[0].FinishReason == openai.FinishReasonLength {
		logger.FromCtx(ctx).Error("llm response truncated by max_tokens",
			"provider", p.name, "model", p.model,
			"elapsed_ms", time.Since(start).Milliseconds())
		return GenerationResult{}, errors.New("llm response truncated by max_tokens")
	}
	content := resp.Choices[0].Message.Content
	if content == "" {
		logger.FromCtx(ctx).Error("llm returned empty content", "provider", p.name, "model", p.model)
		return GenerationResult{}, errors.New("llm returned empty content")
	}

	if !json.Valid([]byte(content)) {
		logger.FromCtx(ctx).Error("llm returned invalid JSON",
			"provider", p.name, "model", p.model,
			"content_len", len(content))
		return GenerationResult{}, errors.New("llm returned invalid JSON")
	}
	result := GenerationResult{Content: content}
	if resp.Usage.PromptTokens != 0 || resp.Usage.CompletionTokens != 0 || resp.Usage.TotalTokens != 0 {
		prompt, completion, total := resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens
		result.Usage = GenerationUsage{PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &total}
	}
	return result, nil
}
