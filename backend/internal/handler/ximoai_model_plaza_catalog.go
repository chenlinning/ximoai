package handler

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type modelPlazaMetadataTarget struct {
	channelIndex int
	sectionIndex int
	modelIndex   int
	input        modelMetadataResolutionInput
}

func (h *AvailableChannelHandler) enrichModelPlazaCatalog(
	ctx context.Context,
	userID int64,
	channels []userAvailableChannel,
	isAdmin bool,
) error {
	if h == nil || h.apiKeyService == nil || h.settingService == nil {
		return fmt.Errorf("model plaza catalog services are unavailable")
	}

	userRates, err := h.apiKeyService.GetUserGroupRates(ctx, userID)
	if err != nil {
		return err
	}
	targets := make([]modelPlazaMetadataTarget, 0)
	lookups := make([]service.ModelMetadataOverride, 0)
	for channelIndex := range channels {
		for sectionIndex := range channels[channelIndex].Platforms {
			section := &channels[channelIndex].Platforms[sectionIndex]
			platform, ok := ximoAIModelPlazaPlatformFor(section.Platform)
			if !ok {
				return fmt.Errorf("model plaza platform %q is unavailable", section.Platform)
			}
			section.DisplayName = platform.DisplayName
			section.Color = platform.Color
			section.Protocol = platform.Protocol
			for groupIndex := range section.Groups {
				if rate, ok := userRates[section.Groups[groupIndex].ID]; ok {
					section.Groups[groupIndex].RateMultiplier = rate
				}
			}
			for modelIndex := range section.SupportedModels {
				model := section.SupportedModels[modelIndex]
				billingModes := make([]string, 0, 1)
				if model.Pricing != nil && model.Pricing.BillingMode != "" {
					billingModes = append(billingModes, model.Pricing.BillingMode)
				}
				input := modelMetadataResolutionInput{
					Platform:         section.Platform,
					Kind:             platform.Kind,
					Protocol:         platform.Protocol,
					Model:            model.Name,
					RecognitionModel: model.recognitionName,
					Capabilities:     platform.Capabilities,
					BillingModes:     billingModes,
				}
				targets = append(targets, modelPlazaMetadataTarget{
					channelIndex: channelIndex,
					sectionIndex: sectionIndex,
					modelIndex:   modelIndex,
					input:        input,
				})
				lookups = append(lookups, service.ModelMetadataOverride{
					Platform: input.Platform,
					Model:    input.Model,
				})
			}
		}
	}

	overrides, loadErr := h.settingService.GetXimoAIModelMetadataOverrides(ctx, lookups)
	if loadErr != nil && isAdmin {
		return fmt.Errorf("load model metadata configuration: %w", loadErr)
	}
	for index, target := range targets {
		var override *service.ModelMetadataOverride
		if loadErr == nil {
			override = overrides[index]
		}
		details := buildModelPlazaMetadata(target.input, override, isAdmin)
		model := &channels[target.channelIndex].Platforms[target.sectionIndex].SupportedModels[target.modelIndex]
		model.Brand = details.Brand
		model.Types = details.Types
		model.InvocationModes = details.InvocationModes
		model.ReasoningLevels = details.ReasoningLevels
		model.ThinkingSupported = details.ThinkingSupported
		model.MetadataEditor = details.Editor
	}
	return nil
}
