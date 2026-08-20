package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (h *GatewayHandler) IsOpenAICompatiblePlatform(ctx context.Context, platform string) bool {
	platform = service.NormalizePlatformSlug(platform)
	switch platform {
	case service.PlatformOpenAI, service.PlatformGrok,
		service.PlatformKimi, service.PlatformZhipu, service.PlatformDeepseek:
		return true
	default:
		return false
	}
}

func (h *GatewayHandler) IsOpenAIChatCompletionsPlatform(ctx context.Context, platform string) bool {
	if h.IsOpenAICompatiblePlatform(ctx, platform) {
		return true
	}
	return h.XimoAIPlatformKind(ctx, platform) == service.PlatformKindOpenAIAudio
}

func (h *GatewayHandler) XimoAIPlatformKind(ctx context.Context, platform string) string {
	return service.XimoAIPlatformKindFromSlug(platform)
}

func (h *GatewayHandler) IsVolcengineAgentPlanPlatform(ctx context.Context, platform string) bool {
	return h.XimoAIPlatformKind(ctx, platform) == service.PlatformKindVolcengineAgentPlan
}

func (h *GatewayHandler) IsXimoAIMediaPlatform(ctx context.Context, platform string) bool {
	return service.IsXimoAIMediaPlatformKind(h.XimoAIPlatformKind(ctx, platform))
}

func (h *GatewayHandler) IsOpenAIAPIKeyProtocolPlatform(ctx context.Context, platform string) bool {
	platform = service.NormalizePlatformSlug(platform)
	return platform == service.PlatformOpenAI || service.IsCNProvider(platform)
}

func (h *GatewayHandler) IsOpenAIImagesPlatform(ctx context.Context, platform string) bool {
	return service.NormalizePlatformSlug(platform) == service.PlatformOpenAI
}

func (h *GatewayHandler) IsOpenAIAudioPlatform(ctx context.Context, platform string) bool {
	return service.NormalizePlatformSlug(platform) == service.PlatformOpenAI
}

func (h *GatewayHandler) isGeminiProtocolAccount(account *service.Account) bool {
	return account != nil && (account.IsGemini() || account.IsGeminiCompatibleAPIKey())
}
