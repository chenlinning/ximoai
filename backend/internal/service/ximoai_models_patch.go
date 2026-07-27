package service

import (
	"context"
	"sort"
	"strings"
)

// GetXimoAIAvailableModels applies the temporary channel-aware /v1/models patch.
// ponytail: Replace this wrapper when upstream exposes channel mapping sources.
func (s *GatewayService) GetXimoAIAvailableModels(ctx context.Context, groupID *int64, platform string) []string {
	if s != nil && s.channelService != nil && groupID != nil {
		channel, err := s.channelService.GetChannelForGroup(ctx, *groupID)
		if err == nil {
			if models, ok := ximoAIChannelMappedModelIDs(channel, platform); ok {
				return models
			}
		}
	}
	if NormalizePlatformSlug(platform) == PlatformVolcengineAgentPlan {
		if models, ok := s.ximoAIVolcengineAgentPlanAccountModelIDs(ctx, groupID); ok {
			return models
		}
	}
	return s.GetAvailableModels(ctx, groupID, platform)
}

func (s *GatewayService) ximoAIVolcengineAgentPlanAccountModelIDs(ctx context.Context, groupID *int64) ([]string, bool) {
	if s == nil || s.accountRepo == nil || groupID == nil {
		return nil, false
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupID(ctx, *groupID)
	if err != nil {
		return nil, false
	}
	modelSet := make(map[string]struct{})
	hasAnyMapping := false
	for _, account := range accounts {
		if account.PlatformRuntimeKind() != PlatformKindVolcengineAgentPlan {
			continue
		}
		mapping := account.GetModelMapping()
		if len(mapping) == 0 {
			continue
		}
		hasAnyMapping = true
		for model := range mapping {
			if model = strings.TrimSpace(model); model != "" {
				modelSet[model] = struct{}{}
			}
		}
	}
	if !hasAnyMapping {
		return nil, false
	}
	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)
	return models, true
}

func ximoAIChannelMappedModelIDs(channel *Channel, platform string) ([]string, bool) {
	if channel == nil {
		return nil, false
	}
	mapping := channel.ModelMapping[NormalizePlatformSlug(platform)]
	if len(mapping) == 0 {
		return nil, false
	}

	models := make([]string, 0, len(mapping))
	for model := range mapping {
		model = strings.TrimSpace(model)
		if model != "" {
			models = append(models, model)
		}
	}
	if len(models) == 0 {
		return nil, false
	}
	sort.Strings(models)
	return models, true
}

// XimoAIModelsFallbackPlatform maps only non-builtin custom platforms to the
// official platform whose default model list matches the selected protocol.
func (s *PlatformService) XimoAIModelsFallbackPlatform(ctx context.Context, slug string) string {
	slug = NormalizePlatformSlug(slug)
	if s == nil || slug == "" {
		return slug
	}
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil || platform == nil || !platform.Enabled {
		return slug
	}
	if platform.IsVolcengineAgentPlan() {
		return PlatformVolcengineAgentPlan
	}
	if platform.Builtin {
		return slug
	}

	switch platform.Protocol {
	case PlatformProtocolOpenAICompatible:
		return PlatformOpenAI
	case PlatformProtocolGemini:
		return PlatformGemini
	case PlatformProtocolAnthropic:
		return PlatformAnthropic
	default:
		return slug
	}
}
