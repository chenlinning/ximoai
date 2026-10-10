package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ximoAIModelPlazaPlatform struct {
	DisplayName  string
	Color        string
	Protocol     string
	Kind         string
	Capabilities []string
}

// ximoAIModelPlazaPresentation contains presentation overrides only. It is
// deliberately not a platform registry: platform existence is checked against
// domain.Platforms below, so old XimoAI platform slugs cannot become routable
// merely by appearing in model-plaza metadata.
var ximoAIModelPlazaPresentation = map[string]ximoAIModelPlazaPlatform{
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
	service.PlatformMiniMax: {
		DisplayName: "MiniMax", Color: "#F43F5E", Protocol: "openai_compatible",
		Capabilities: []string{"chat_completions"},
	},
	service.PlatformOpenCodeGo: {
		DisplayName: "OpenCode", Color: "#F59E0B", Protocol: "openai_compatible",
		Capabilities: []string{"chat_completions"},
	},
	service.PlatformTypeSafe: {
		DisplayName: "TypeSafe / Jev", Color: "#0EA5E9", Protocol: "native",
		Capabilities: []string{"messages"},
	},
	service.PlatformCommandCode: {
		DisplayName: "Command Code", Color: "#737373", Protocol: "openai_compatible",
		Capabilities: []string{"chat_completions"},
	},
	service.PlatformCline: {
		DisplayName: "Cline", Color: "#8B5CF6", Protocol: "openai_compatible",
		Capabilities: []string{"chat_completions"},
	},
}

func ximoAIModelPlazaPlatformFor(slug string) (ximoAIModelPlazaPlatform, bool) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	spec, ok := domain.LookupPlatform(slug)
	if !ok {
		return ximoAIModelPlazaPlatform{}, false
	}
	if platform, ok := ximoAIModelPlazaPresentation[slug]; ok {
		return platform, true
	}

	// New upstream platforms remain usable by the custom plaza without adding
	// another XimoAI platform table. Their coarse capabilities come from the
	// upstream gateway; the registry still supplies model-specific metadata
	// when an audited record exists.
	return ximoAIModelPlazaPlatform{
		DisplayName:  spec.DisplayName,
		Color:        "#64748B",
		Protocol:     modelPlazaProtocolForGateway(spec.Gateway),
		Capabilities: modelPlazaCapabilitiesFor(spec),
	}, true
}

func modelPlazaProtocolForGateway(gateway domain.PlatformGateway) string {
	switch gateway {
	case domain.PlatformGatewayAnthropic,
		domain.PlatformGatewayGemini,
		domain.PlatformGatewayAntigravity:
		return "native"
	default:
		return "openai_compatible"
	}
}

func modelPlazaCapabilitiesFor(spec domain.PlatformSpec) []string {
	switch spec.Gateway {
	case domain.PlatformGatewayAnthropic:
		return []string{"messages"}
	case domain.PlatformGatewayGemini, domain.PlatformGatewayAntigravity:
		return []string{"messages", "native_gemini"}
	case domain.PlatformGatewayOpenAI:
		return []string{"chat_completions"}
	default:
		return nil
	}
}
