package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// XimoAIAvailableChannel keeps recognition metadata outside the native channel contract.
type XimoAIAvailableChannel struct {
	AvailableChannel
	ModelMapping map[string]map[string]string
}

func (c XimoAIAvailableChannel) RecognitionName(platform, name string) string {
	for source, target := range c.ModelMapping[platform] {
		if strings.EqualFold(strings.TrimSpace(source), strings.TrimSpace(name)) &&
			strings.TrimSpace(target) != "" && !strings.ContainsAny(target, "*?") {
			return strings.TrimSpace(target)
		}
	}
	return name
}

// ListAvailableChannelPricing limits the custom catalog to channel-configured prices.
func (s *ChannelService) ListAvailableChannelPricing(ctx context.Context) ([]XimoAIAvailableChannel, error) {
	channels, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}
	groupByID := make(map[int64]AvailableGroupRef, len(groups))
	for i := range groups {
		g := groups[i]
		groupByID[g.ID] = AvailableGroupRef{
			ID:                 g.ID,
			Name:               g.Name,
			I18n:               g.I18n,
			Platform:           g.Platform,
			SubscriptionType:   g.SubscriptionType,
			RateMultiplier:     g.RateMultiplier,
			PeakRateEnabled:    g.PeakRateEnabled,
			PeakStart:          g.PeakStart,
			PeakEnd:            g.PeakEnd,
			PeakRateMultiplier: g.PeakRateMultiplier,
			IsExclusive:        g.IsExclusive,
		}
	}

	out := make([]XimoAIAvailableChannel, 0, len(channels))
	for i := range channels {
		ch := &channels[i]
		groups := make([]AvailableGroupRef, 0, len(ch.GroupIDs))
		for _, gid := range ch.GroupIDs {
			if ref, ok := groupByID[gid]; ok {
				groups = append(groups, ref)
			}
		}
		sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })

		ch.normalizeBillingModelSource()

		supported := ch.SupportedModels()
		supported = filterXimoAIModelPlazaModels(ch, supported)

		out = append(out, XimoAIAvailableChannel{AvailableChannel: AvailableChannel{
			ID:                 ch.ID,
			Name:               ch.Name,
			Description:        ch.Description,
			Status:             ch.Status,
			BillingModelSource: ch.BillingModelSource,
			RestrictModels:     ch.RestrictModels,
			Groups:             groups,
			SupportedModels:    supported,
		}, ModelMapping: ch.ModelMapping})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}
