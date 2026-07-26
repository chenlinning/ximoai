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
	registered := h.registeredPlatform(ctx, platform)
	return registered != nil && !service.IsXimoAIMediaPlatformKind(registered.RuntimeKind()) && registered.IsOpenAICompatible()
}

func (h *GatewayHandler) IsOpenAIChatCompletionsPlatform(ctx context.Context, platform string) bool {
	if h.IsOpenAICompatiblePlatform(ctx, platform) {
		return true
	}
	return h.XimoAIPlatformKind(ctx, platform) == service.PlatformKindOpenAIAudio
}

func (h *GatewayHandler) XimoAIPlatformKind(ctx context.Context, platform string) string {
	platform = service.NormalizePlatformSlug(platform)
	if registered := h.registeredPlatform(ctx, platform); registered != nil {
		if registered.IsVolcengineAgentPlan() {
			return service.PlatformKindVolcengineAgentPlan
		}
		return registered.RuntimeKind()
	}
	return service.XimoAIPlatformKindFromLegacySlug(platform)
}

func (h *GatewayHandler) IsVolcengineAgentPlanPlatform(ctx context.Context, platform string) bool {
	return h.XimoAIPlatformKind(ctx, platform) == service.PlatformKindVolcengineAgentPlan
}

func (h *GatewayHandler) IsXimoAIMediaPlatform(ctx context.Context, platform string) bool {
	return service.IsXimoAIMediaPlatformKind(h.XimoAIPlatformKind(ctx, platform))
}

func (h *GatewayHandler) registeredPlatform(ctx context.Context, platform string) *service.Platform {
	if platform == "" || h == nil || h.platformService == nil {
		return nil
	}
	registered, err := h.platformService.GetBySlug(ctx, platform)
	if err != nil || registered == nil || !registered.Enabled {
		return nil
	}
	return registered
}

func (h *GatewayHandler) IsOpenAIAPIKeyProtocolPlatform(ctx context.Context, platform string) bool {
	platform = service.NormalizePlatformSlug(platform)
	if platform == service.PlatformOpenAI {
		return true
	}
	if platform == "" || h == nil || h.platformService == nil {
		return false
	}
	registered := h.registeredPlatform(ctx, platform)
	if registered == nil || service.IsXimoAIMediaPlatformKind(registered.RuntimeKind()) {
		return false
	}
	return registered.Protocol == service.PlatformProtocolOpenAI ||
		(!registered.Builtin && registered.Protocol == service.PlatformProtocolOpenAICompatible)
}

func (h *GatewayHandler) isGeminiProtocolAccount(account *service.Account) bool {
	return account != nil && (account.IsGemini() || account.IsGeminiCompatibleAPIKey())
}
