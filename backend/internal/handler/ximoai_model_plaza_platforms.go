package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ximoAIModelPlazaPlatform struct {
	DisplayName  string
	Color        string
	Protocol     string
	Kind         string
	Capabilities []string
}

var ximoAIModelPlazaPlatforms = map[string]ximoAIModelPlazaPlatform{
	service.PlatformAnthropic: {
		DisplayName: "Anthropic", Color: "#D97706", Protocol: "native",
		Capabilities: []string{"messages"},
	},
	service.PlatformOpenAI: {
		DisplayName: "OpenAI", Color: "#10A37F", Protocol: "openai",
		Capabilities: []string{"responses", "chat_completions", "embeddings", "images", "audio", "realtime", "codex"},
	},
	service.PlatformGemini: {
		DisplayName: "Gemini", Color: "#4285F4", Protocol: "native",
		Capabilities: []string{"messages", "native_gemini", "videos"},
	},
	service.PlatformAntigravity: {
		DisplayName: "Antigravity", Color: "#7C3AED", Protocol: "native",
		Capabilities: []string{"messages", "native_gemini"},
	},
	service.PlatformGrok: {
		DisplayName: "Grok", Color: "#111827", Protocol: "openai_compatible",
		Capabilities: []string{"responses", "chat_completions", "images", "videos"},
	},
	service.PlatformKimi: {
		DisplayName: "Kimi", Color: "#EC4899", Protocol: "openai_compatible",
		Capabilities: []string{"chat_completions", "messages"},
	},
	service.PlatformZhipu: {
		DisplayName: "Zhipu GLM", Color: "#6366F1", Protocol: "openai_compatible",
		Capabilities: []string{"chat_completions", "messages"},
	},
	service.PlatformDeepseek: {
		DisplayName: "DeepSeek", Color: "#14B8A6", Protocol: "openai_compatible",
		Capabilities: []string{"responses", "chat_completions", "messages"},
	},
	service.PlatformGrokVideo: {
		DisplayName: "Grok Video", Color: "#111827", Protocol: "openai_compatible", Kind: service.PlatformKindGrokVideo,
		Capabilities: []string{"videos"},
	},
	service.PlatformOpenAIAudio: {
		DisplayName: "OpenAI Audio", Color: "#0F766E", Protocol: "openai_compatible", Kind: service.PlatformKindOpenAIAudio,
		Capabilities: []string{"chat_completions", "audio"},
	},
	service.PlatformKlingAudio: {
		DisplayName: "Kling Audio", Color: "#0EA5E9", Protocol: "openai_compatible", Kind: service.PlatformKindKlingAudio,
		Capabilities: []string{"audio"},
	},
	service.PlatformVolcengineAgentPlan: {
		DisplayName: "Volcengine Agent Plan", Color: "#E5484D", Protocol: "native", Kind: service.PlatformKindVolcengineAgentPlan,
		Capabilities: []string{"images", "audio"},
	},
}

func ximoAIModelPlazaPlatformFor(slug string) (ximoAIModelPlazaPlatform, bool) {
	platform, ok := ximoAIModelPlazaPlatforms[strings.ToLower(strings.TrimSpace(slug))]
	return platform, ok
}
