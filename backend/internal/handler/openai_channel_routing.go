package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func openAIResolvedRoutingModel(ctx context.Context, requestModel string) string {
	if resolvedModel, ok := service.ResolvedUpstreamModelFromContext(ctx); ok {
		return strings.TrimSpace(resolvedModel)
	}
	return strings.TrimSpace(requestModel)
}

func (h *OpenAIGatewayHandler) resolveCompositeRouteContext(
	ctx context.Context,
	apiKey *service.APIKey,
	model string,
	endpoint string,
) (context.Context, error) {
	if apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite || h == nil || h.compositeResolver == nil {
		return ctx, nil
	}
	if _, ok := service.ResolvedTargetPlatformFromContext(ctx); ok {
		return ctx, nil
	}
	decision, err := h.compositeResolver.Resolve(ctx, apiKey.Group.ID, model, endpoint)
	if err != nil {
		return ctx, err
	}
	if !decision.Matched {
		return ctx, nil
	}
	return service.WithCompositeRouteDecision(ctx, decision), nil
}

func openAIChannelRoutingModel(requestModel string, channelMapping service.ChannelMappingResult) string {
	if channelMapping.Mapped && strings.TrimSpace(channelMapping.MappedModel) != "" {
		return strings.TrimSpace(channelMapping.MappedModel)
	}
	return requestModel
}

func openAIRoutedBody(body []byte, requestModel, routingModel string, replace openAIModelBodyReplaceFunc) []byte {
	requestModel = strings.TrimSpace(requestModel)
	routingModel = strings.TrimSpace(routingModel)
	if replace == nil || routingModel == "" || routingModel == requestModel {
		return body
	}
	return replace(body, routingModel)
}

func validateOpenAIImageRoutingModel(platform, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("images endpoint requires an image model")
	}
	if service.NormalizePlatformSlug(platform) == service.PlatformOpenAI && !service.IsGPTImageGenerationModel(model) {
		return fmt.Errorf("images endpoint requires an image model, got %q", model)
	}
	return nil
}
