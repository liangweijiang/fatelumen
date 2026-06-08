package llm

import "context"

// LLMProvider 抽象所有大模型调用。P5 原则：业务层只依赖此接口。
type LLMProvider interface {
	// GenerateJSON 给定 system + user prompt，返回严格 JSON 字符串。
	// 必须开启 provider 的 JSON mode / structured output。
	GenerateJSON(ctx context.Context, system, user string, opts ...Option) (string, error)
	Name() string
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
