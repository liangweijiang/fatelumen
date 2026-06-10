package llm

import "fatelumen/backend/internal/pkg/logger"

func NewDeepSeekProvider(apiKey, baseURL, model string, log *logger.Logger) LLMProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return newOpenAICompat("deepseek", apiKey, baseURL, model, log)
}
