package llm

func NewDeepSeekProvider(apiKey, baseURL, model string) LLMProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return newOpenAICompat("deepseek", apiKey, baseURL, model)
}
