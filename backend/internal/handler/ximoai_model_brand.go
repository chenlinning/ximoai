package handler

import "strings"

const ximoAIModelBrandOther = "Other"

type modelBrandRule struct {
	brand    string
	prefixes []string
}

var ximoAIModelBrandRules = []modelBrandRule{
	{brand: "OpenAI", prefixes: []string{"openai", "gpt", "chatgpt", "o1", "o3", "o4", "dall-e", "sora", "whisper", "tts-1", "text-embedding-3", "codex"}},
	{brand: "Anthropic", prefixes: []string{"anthropic", "claude"}},
	{brand: "Google", prefixes: []string{"google", "gemini", "imagen", "veo"}},
	{brand: "xAI", prefixes: []string{"xai", "grok"}},
	{brand: "DeepSeek", prefixes: []string{"deepseek"}},
	{brand: "Qwen", prefixes: []string{"qwen", "qwq", "wan"}},
	{brand: "Doubao", prefixes: []string{"doubao", "seedream", "seedance"}},
	{brand: "Moonshot AI", prefixes: []string{"kimi", "moonshot"}},
	{brand: "Zhipu AI", prefixes: []string{"zhipu", "glm", "cogview", "cogvideo"}},
	{brand: "MiniMax", prefixes: []string{"minimax"}},
	{brand: "Meta", prefixes: []string{"meta", "llama"}},
	{brand: "Mistral AI", prefixes: []string{"mistral", "mixtral"}},
	{brand: "Cohere", prefixes: []string{"cohere", "command-r"}},
	{brand: "Baidu", prefixes: []string{"baidu", "ernie"}},
	{brand: "iFlytek", prefixes: []string{"iflytek", "spark"}},
	{brand: "Tencent", prefixes: []string{"tencent", "hunyuan"}},
	{brand: "Perplexity", prefixes: []string{"perplexity", "sonar"}},
	{brand: "Kuaishou", prefixes: []string{"kuaishou", "kling", "keling"}},
}

func detectXimoAIModelBrand(model string) string {
	for _, candidate := range ximoAIModelBrandCandidates(model) {
		for _, rule := range ximoAIModelBrandRules {
			for _, prefix := range rule.prefixes {
				if hasXimoAIModelBrandPrefix(candidate, prefix) {
					return rule.brand
				}
			}
		}
	}
	return ximoAIModelBrandOther
}

func ximoAIModelBrandCandidates(model string) []string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return nil
	}
	parts := strings.FieldsFunc(normalized, func(value rune) bool {
		return value == '/' || value == ':'
	})
	candidates := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "models" {
			candidates = append(candidates, part)
		}
	}
	if len(candidates) == 0 {
		return []string{normalized}
	}
	return candidates
}

func hasXimoAIModelBrandPrefix(model, prefix string) bool {
	if model == prefix {
		return true
	}
	if !strings.HasPrefix(model, prefix) || len(model) == len(prefix) {
		return false
	}
	switch model[len(prefix)] {
	case '-', '_', '.':
		return true
	default:
		return model[len(prefix)] >= '0' && model[len(prefix)] <= '9'
	}
}
