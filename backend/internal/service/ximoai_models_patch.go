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
	return s.GetAvailableModels(ctx, groupID, platform)
}

func ximoAIChannelMappedModelIDs(channel *Channel, platform string) ([]string, bool) {
	if channel == nil {
		return nil, false
	}
	mapping := channel.ModelMapping[strings.ToLower(strings.TrimSpace(platform))]
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
