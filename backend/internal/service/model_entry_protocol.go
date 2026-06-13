package service

import (
	"context"
	"strings"
)

type GatewayPricedModelDetail struct {
	Name     string
	Platform string
	Pricing  *ChannelModelPricing
}

func (s *GatewayService) GetPricedModelDetails(ctx context.Context, groupID *int64, platform string) []GatewayPricedModelDetail {
	if s == nil || s.channelService == nil || groupID == nil {
		return nil
	}

	ch, err := s.channelService.GetChannelForGroup(ctx, *groupID)
	if err != nil || ch == nil {
		return nil
	}

	platform = NormalizePlatformSlug(platform)
	supported := filterSupportedModelsWithPricing(ch.SupportedModels())
	seen := make(map[string]struct{}, len(supported))
	models := make([]GatewayPricedModelDetail, 0, len(supported))
	for _, model := range supported {
		modelPlatform := NormalizePlatformSlug(model.Platform)
		if platform != "" && modelPlatform != platform {
			continue
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		models = append(models, GatewayPricedModelDetail{
			Name:     name,
			Platform: modelPlatform,
			Pricing:  model.Pricing,
		})
	}
	return models
}

func (s *GatewayService) ResolveEffectiveGroupRateMultiplier(ctx context.Context, userID, groupID int64, groupDefaultMultiplier float64) float64 {
	return s.getUserGroupRateMultiplier(ctx, userID, groupID, groupDefaultMultiplier)
}
