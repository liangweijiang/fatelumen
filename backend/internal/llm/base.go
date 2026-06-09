package llm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type openAICompatProvider struct {
	name    string
	client  *openai.Client
	model   string
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

func (p *openAICompatProvider) Name() string {
	return p.name
}

func (p *openAICompatProvider) GenerateJSON(ctx context.Context, system, user string, opts ...Option) (string, error) {
	cc := &callConfig{temperature: 0.7, maxTokens: 600}
	for _, o := range opts {
		o(cc)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
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
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("llm returned empty response")
	}
	content := resp.Choices[0].Message.Content
	if content == "" {
		return "", errors.New("llm returned empty content")
	}

	if !json.Valid([]byte(content)) {
		return "", errors.New("llm returned invalid JSON")
	}
	return content, nil
}
