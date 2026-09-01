package llm

// ProviderPreset and ModelPreset are form-fill helpers. They do not control
// runtime routing; persisted configurations do.
type ProviderPreset struct {
	Code    string        `json:"code"`
	Name    string        `json:"name"`
	BaseURL string        `json:"base_url"`
	Models  []ModelPreset `json:"models"`
}

type ModelPreset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProviderCatalog centralizes the mainstream text models usable for reports.
// Custom providers and model IDs remain supported by the admin API.
func ProviderCatalog() []ProviderPreset {
	return []ProviderPreset{
		{Code: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com", Models: []ModelPreset{{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash"}, {ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro"}, {ID: "deepseek-chat", Name: "DeepSeek Chat（兼容别名）"}, {ID: "deepseek-reasoner", Name: "DeepSeek Reasoner（兼容别名）"}}},
		{Code: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", Models: []ModelPreset{{ID: "gpt-5.6", Name: "GPT-5.6"}, {ID: "gpt-5.6-sol", Name: "GPT-5.6 Sol"}, {ID: "gpt-5.6-terra", Name: "GPT-5.6 Terra"}, {ID: "gpt-5.6-luna", Name: "GPT-5.6 Luna"}, {ID: "gpt-5.5", Name: "GPT-5.5"}, {ID: "gpt-5.4", Name: "GPT-5.4"}, {ID: "gpt-5.4-mini", Name: "GPT-5.4 mini"}}},
		{Code: "anthropic", Name: "Anthropic", BaseURL: "https://api.anthropic.com/v1", Models: []ModelPreset{{ID: "claude-opus-5", Name: "Claude Opus 5"}, {ID: "claude-sonnet-5", Name: "Claude Sonnet 5"}, {ID: "claude-fable-5", Name: "Claude Fable 5"}, {ID: "claude-opus-4-8", Name: "Claude Opus 4.8"}, {ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6"}, {ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5"}}},
		{Code: "google", Name: "Google Gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Models: []ModelPreset{{ID: "gemini-3.6-flash", Name: "Gemini 3.6 Flash"}, {ID: "gemini-3.5-flash", Name: "Gemini 3.5 Flash"}, {ID: "gemini-3.5-flash-lite", Name: "Gemini 3.5 Flash-Lite"}, {ID: "gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash-Lite"}}},
		{Code: "qwen", Name: "阿里云百炼", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Models: []ModelPreset{{ID: "qwen-max", Name: "通义千问 Max"}, {ID: "qwen-plus", Name: "通义千问 Plus"}, {ID: "qwen-turbo", Name: "通义千问 Turbo"}, {ID: "qwq-plus", Name: "通义千问 QwQ Plus"}}},
		{Code: "doubao", Name: "火山方舟", BaseURL: "https://ark.cn-beijing.volces.com/api/v3", Models: []ModelPreset{{ID: "doubao-seed-1-6", Name: "豆包 Seed 1.6"}, {ID: "doubao-1-5-pro-32k", Name: "豆包 1.5 Pro 32K"}, {ID: "deepseek-v3-250324", Name: "DeepSeek V3（方舟）"}, {ID: "deepseek-r1-250120", Name: "DeepSeek R1（方舟）"}}},
		{Code: "zhipu", Name: "智谱", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Models: []ModelPreset{{ID: "glm-4.5", Name: "GLM-4.5"}, {ID: "glm-4-plus", Name: "GLM-4 Plus"}, {ID: "glm-4-air", Name: "GLM-4 Air"}, {ID: "glm-z1-air", Name: "GLM-Z1 Air"}}},
		{Code: "moonshot", Name: "月之暗面 Kimi", BaseURL: "https://api.moonshot.cn/v1", Models: []ModelPreset{{ID: "kimi-k2", Name: "Kimi K2"}, {ID: "moonshot-v1-128k", Name: "Moonshot V1 128K"}, {ID: "moonshot-v1-32k", Name: "Moonshot V1 32K"}, {ID: "moonshot-v1-8k", Name: "Moonshot V1 8K"}}},
		{Code: "minimax", Name: "MiniMax", BaseURL: "https://api.minimax.chat/v1", Models: []ModelPreset{{ID: "MiniMax-M1", Name: "MiniMax M1"}, {ID: "MiniMax-Text-01", Name: "MiniMax Text 01"}}},
		{Code: "hunyuan", Name: "腾讯混元", BaseURL: "https://api.hunyuan.cloud.tencent.com/v1", Models: []ModelPreset{{ID: "hunyuan-turbos-latest", Name: "混元 TurboS"}, {ID: "hunyuan-turbo-latest", Name: "混元 Turbo"}, {ID: "hunyuan-large", Name: "混元 Large"}}},
		{Code: "ernie", Name: "百度千帆", BaseURL: "https://qianfan.baidubce.com/v2", Models: []ModelPreset{{ID: "ernie-4.5-8k-preview", Name: "文心 ERNIE 4.5"}, {ID: "ernie-x1-turbo-32k", Name: "文心 ERNIE X1 Turbo"}, {ID: "ernie-speed-128k", Name: "文心 ERNIE Speed 128K"}}},
		{Code: "siliconflow", Name: "硅基流动", BaseURL: "https://api.siliconflow.cn/v1", Models: []ModelPreset{{ID: "deepseek-ai/DeepSeek-V3", Name: "DeepSeek V3"}, {ID: "deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1"}, {ID: "Qwen/Qwen3-235B-A22B", Name: "Qwen3 235B"}}},
		{Code: "openrouter", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", Models: []ModelPreset{{ID: "openai/gpt-5.6", Name: "GPT-5.6"}, {ID: "anthropic/claude-sonnet-5", Name: "Claude Sonnet 5"}, {ID: "google/gemini-3.6-flash", Name: "Gemini 3.6 Flash"}, {ID: "deepseek/deepseek-v4-pro", Name: "DeepSeek V4 Pro"}}},
	}
}
