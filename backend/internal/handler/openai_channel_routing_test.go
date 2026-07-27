package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type openAICompositeRouteRepoStub struct {
	routes []service.CompositeModelRoute
}

func (s openAICompositeRouteRepoStub) ListByGroup(_ context.Context, _ int64, _ bool) ([]service.CompositeModelRoute, error) {
	return s.routes, nil
}

func (openAICompositeRouteRepoStub) Create(context.Context, *service.CompositeModelRoute) error {
	return nil
}

func (openAICompositeRouteRepoStub) Update(context.Context, *service.CompositeModelRoute) error {
	return nil
}

func (openAICompositeRouteRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (openAICompositeRouteRepoStub) DeleteByGroup(context.Context, int64) error {
	return nil
}

func TestOpenAIResolvedRoutingModelUsesCompositeThenChannelMapping(t *testing.T) {
	ctx := service.WithCompositeRouteDecision(context.Background(), service.CompositeRouteDecision{
		Matched:        true,
		TargetPlatform: service.PlatformOpenAI,
		PublicModel:    "draw-pro",
		UpstreamModel:  "gpt-image-2",
	})
	mapping := service.ChannelMappingResult{Mapped: true, MappedModel: "gpt-image-2-custom"}

	baseModel := openAIResolvedRoutingModel(ctx, "draw-pro")
	routingModel := openAIChannelRoutingModel(baseModel, mapping)

	require.Equal(t, "gpt-image-2", baseModel)
	require.Equal(t, "gpt-image-2-custom", routingModel)
}

func TestOpenAICompatibleRequestPlatformPreservesCompositeCustomPlatform(t *testing.T) {
	ctx := service.WithCompositeRouteDecision(context.Background(), service.CompositeRouteDecision{
		Matched:        true,
		TargetPlatform: "acme-openai",
		PublicModel:    "public-chat",
		UpstreamModel:  "acme-chat-v2",
	})

	require.Equal(t, "acme-openai", openAICompatibleRequestPlatform(ctx, nil))
}

func TestResolveOpenAICompositeRouteContextUsesExplicitCustomRoute(t *testing.T) {
	resolver := service.NewCompositeRouteResolver(openAICompositeRouteRepoStub{routes: []service.CompositeModelRoute{
		{
			ID:             1,
			GroupID:        9,
			PublicModel:    "public-acme",
			MatchType:      service.CompositeRouteMatchExact,
			TargetPlatform: "acme-openai",
			UpstreamModel:  "acme-reasoning",
			Endpoint:       service.CompositeRouteEndpointResponses,
			Enabled:        true,
		},
	}})
	h := &OpenAIGatewayHandler{compositeResolver: resolver}
	apiKey := &service.APIKey{Group: &service.Group{ID: 9, Platform: service.PlatformComposite}}

	ctx, err := h.resolveCompositeRouteContext(
		context.Background(),
		apiKey,
		"public-acme",
		service.CompositeRouteEndpointResponses,
	)

	require.NoError(t, err)
	platform, ok := service.ResolvedTargetPlatformFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "acme-openai", platform)
	model, ok := service.ResolvedUpstreamModelFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "acme-reasoning", model)
}

func TestOpenAIRoutedBodyRewritesCompositeModelWithoutChannelMapping(t *testing.T) {
	body := []byte(`{"model":"draw-pro","input":"draw"}`)
	got := openAIRoutedBody(body, "draw-pro", "gpt-image-2", service.ReplaceModelInBody)

	require.JSONEq(t, `{"model":"gpt-image-2","input":"draw"}`, string(got))
}

func TestOpenAIResponsesRequiredCapabilityRequiresResponsesForCustomImagePlatform(t *testing.T) {
	require.Equal(
		t,
		service.OpenAIEndpointCapabilityResponses,
		openAIResponsesRequiredCapability(true, "custom-openai"),
	)
	require.Equal(
		t,
		service.OpenAIEndpointCapabilityChatCompletions,
		openAIResponsesRequiredCapability(true, service.PlatformGrok),
	)
}

func TestValidateOpenAIImageRoutingModelUsesFinalPlatform(t *testing.T) {
	require.NoError(t, validateOpenAIImageRoutingModel(service.PlatformOpenAI, "gpt-image-2"))
	require.Error(t, validateOpenAIImageRoutingModel(service.PlatformOpenAI, "draw-pro"))
	require.NoError(t, validateOpenAIImageRoutingModel("custom-openai", "flux-1.1-pro"))
	require.Error(t, validateOpenAIImageRoutingModel("custom-openai", ""))
}
