package service

import (
	"context"
	"strings"
)

// VolcengineAgentPlanModelRoute carries the user-visible model and the
// channel-facing model used while selecting a native Volcengine account.
// Account mappings are resolved only after an account has been selected.
type VolcengineAgentPlanModelRoute struct {
	PublicModel        string
	ChannelMappedModel string
}

type volcengineAgentPlanModelRouteContextKey struct{}

// WithVolcengineAgentPlanModelRoute keeps native model mapping local to the
// Volcengine Agent Plan request path without changing generic gateway APIs.
func WithVolcengineAgentPlanModelRoute(ctx context.Context, route VolcengineAgentPlanModelRoute) context.Context {
	return context.WithValue(ctx, volcengineAgentPlanModelRouteContextKey{}, route)
}

func VolcengineAgentPlanModelRouteFromContext(ctx context.Context) (VolcengineAgentPlanModelRoute, bool) {
	if ctx == nil {
		return VolcengineAgentPlanModelRoute{}, false
	}
	route, ok := ctx.Value(volcengineAgentPlanModelRouteContextKey{}).(VolcengineAgentPlanModelRoute)
	if !ok {
		return VolcengineAgentPlanModelRoute{}, false
	}
	route.PublicModel = strings.TrimSpace(route.PublicModel)
	route.ChannelMappedModel = strings.TrimSpace(route.ChannelMappedModel)
	return route, route.PublicModel != "" || route.ChannelMappedModel != ""
}

// ResolveVolcengineAgentPlanUpstreamModel applies only explicit account
// mapping. It deliberately does not infer aliases from model name prefixes.
func ResolveVolcengineAgentPlanUpstreamModel(account *Account, channelMappedModel string) string {
	model := strings.TrimSpace(channelMappedModel)
	if model == "" {
		return ""
	}
	if account == nil {
		return model
	}
	return strings.TrimSpace(account.GetMappedModel(model))
}

// VolcengineAgentPlanAccountSupportsModel accepts either the public model or
// the channel-mapped model. This is needed when a channel exposes an alias but
// an account whitelist contains the canonical upstream model (and vice versa).
// No provider-specific model-name recognition is performed here.
func VolcengineAgentPlanAccountSupportsModel(account *Account, publicModel, channelMappedModel string) bool {
	if account == nil {
		return false
	}
	if len(account.GetModelMapping()) == 0 {
		return true
	}
	for _, model := range []string{publicModel, channelMappedModel} {
		model = strings.TrimSpace(model)
		if model != "" && account.IsModelSupported(model) {
			return true
		}
	}
	return false
}

// volcengineAgentPlanModelChainRestrictedByChannel keeps the native route
// allowlist aligned with the same model candidates used by usage billing.
func (s *GatewayService) volcengineAgentPlanModelChainRestrictedByChannel(
	ctx context.Context,
	groupID int64,
	primaryModel string,
	route VolcengineAgentPlanModelRoute,
) bool {
	if s == nil || s.channelService == nil {
		return false
	}
	models := usageBillingModelCandidates(primaryModel, route.ChannelMappedModel, route.PublicModel)
	for _, model := range models {
		if !s.channelService.IsModelRestricted(ctx, groupID, model) {
			return false
		}
	}
	return len(models) > 0
}
