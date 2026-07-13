package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (h *GatewayHandler) IsOpenAICompatiblePlatform(ctx context.Context, platform string) bool {
	platform = service.NormalizePlatformSlug(platform)
	switch platform {
	case service.PlatformOpenAI, service.PlatformGrok:
		return true
	}
	return h != nil && h.platformService != nil && h.platformService.IsOpenAICompatible(ctx, platform)
}
