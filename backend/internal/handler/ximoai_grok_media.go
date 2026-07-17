package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

func grokMediaPlatformForAPIKey(apiKey *service.APIKey) string {
	if openAIPlatformForAPIKey(apiKey) == service.PlatformGrokVideo {
		return service.PlatformGrokVideo
	}
	return service.PlatformGrok
}
