package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func grokMediaPlatformForAPIKey(ctx context.Context, apiKey *service.APIKey) string {
	if platform, ok := service.ResolvedTargetPlatformFromContext(ctx); ok && platform != "" {
		return platform
	}
	platform := openAIPlatformForAPIKey(apiKey)
	if platform == "" {
		return service.PlatformGrok
	}
	return platform
}
