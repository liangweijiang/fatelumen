package llm

import "context"

// LLMProvider 抽象所有大模型调用。P5 原则：业务层只依赖此接口。
type LLMProvider interface {
	// GenerateJSON 给定 system + user prompt，返回严格 JSON 字符串。
	// 必须开启 provider 的 JSON mode / structured output。
	GenerateJSON(ctx context.Context, system, user string, opts ...Option) (string, error)
	Name() string
}

// GenerationUsage uses pointers so a provider that omits usage is stored as
// unavailable instead of being mistaken for a measured zero.
type GenerationUsage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	TotalTokens      *int `json:"total_tokens,omitempty"`
}

type GenerationResult struct {
	Content string          `json:"content"`
	Usage   GenerationUsage `json:"usage"`
}

// DetailedLLMProvider is optional so existing providers and tests remain
// source-compatible while report execution can capture provider usage.
type DetailedLLMProvider interface {
	LLMProvider
	GenerateJSONDetailed(ctx context.Context, system, user string, opts ...Option) (GenerationResult, error)
}

func GenerateJSONDetailed(ctx context.Context, provider LLMProvider, system, user string, opts ...Option) (GenerationResult, error) {
	if detailed, ok := provider.(DetailedLLMProvider); ok {
		return detailed.GenerateJSONDetailed(ctx, system, user, opts...)
	}
	content, err := provider.GenerateJSON(ctx, system, user, opts...)
	return GenerationResult{Content: content}, err
}

type callConfig struct {
	temperature float32
	maxTokens   int
}

type Option func(*callConfig)

func WithTemperature(t float32) Option {
	return func(cc *callConfig) { cc.temperature = t }
}

func WithMaxTokens(n int) Option {
	return func(cc *callConfig) { cc.maxTokens = n }
}
