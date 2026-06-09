package llm

func NewOpenAIProvider(apiKey, model string) LLMProvider {
	baseURL := "https://api.openai.com/v1"
	if model == "" {
		model = "gpt-4o-mini"
	}
	return newOpenAICompat("openai", apiKey, baseURL, model)
}
