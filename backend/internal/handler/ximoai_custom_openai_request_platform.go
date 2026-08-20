package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

func ximoAICustomOpenAIRequestPlatform(apiKey *service.APIKey) string {
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	platform := service.NormalizePlatformSlug(apiKey.Group.Platform)
	switch platform {
	case service.PlatformGrokVideo, service.PlatformOpenAIAudio, service.PlatformKlingAudio:
		return platform
	default:
		return ""
	}
}
