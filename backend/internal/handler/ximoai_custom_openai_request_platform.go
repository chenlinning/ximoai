package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

func ximoAICustomOpenAIRequestPlatform(apiKey *service.APIKey) string {
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	platform := service.NormalizePlatformSlug(apiKey.Group.Platform)
	switch platform {
	case "", service.PlatformOpenAI, service.PlatformGrok, service.PlatformAnthropic,
		service.PlatformGemini, service.PlatformAntigravity:
		return ""
	default:
		return platform
	}
}
