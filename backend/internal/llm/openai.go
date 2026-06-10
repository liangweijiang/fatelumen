package llm

import "fatelumen/backend/internal/pkg/logger"

func NewOpenAIProvider(apiKey, model string, log *logger.Logger) LLMProvider {
	baseURL := "https://api.openai.com/v1"
	if model == "" {
		model = "gpt-4o-mini"
	}
	return newOpenAICompat("openai", apiKey, baseURL, model, log)
}
