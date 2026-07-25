package handler

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type modelPlazaModelDetails struct {
	Types            []string
	Capabilities     []string
	Protocols        []string
	InvocationModes  []string
	APIDocumentation modelAPIDocsResponse
}

type modelPlazaCatalogTarget struct {
	channelIndex int
	sectionIndex int
	modelIndex   int
	input        modelAPIDocsResolutionInput
}

func (h *AvailableChannelHandler) enrichModelPlazaCatalog(
	ctx context.Context,
	userID int64,
	channels []userAvailableChannel,
	isAdmin bool,
) error {
	if h == nil || h.apiKeyService == nil || h.settingService == nil || h.platformService == nil {
		return fmt.Errorf("model plaza catalog services are unavailable")
	}

	userRates, err := h.apiKeyService.GetUserGroupRates(ctx, userID)
	if err != nil {
		return err
	}
	platforms, err := h.platformService.List(ctx, true)
	if err != nil {
		return err
	}
	platformIndex := make(map[string]service.Platform, len(platforms))
	for _, platform := range platforms {
		platformIndex[platform.Slug] = platform
	}

	targets := make([]modelPlazaCatalogTarget, 0)
	lookupTargets := make([]service.ModelAPIDocsBinding, 0)
	for channelIndex := range channels {
		for sectionIndex := range channels[channelIndex].Platforms {
			section := &channels[channelIndex].Platforms[sectionIndex]
			platform, ok := platformIndex[section.Platform]
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
				input := normalizeModelAPIDocsResolutionInput(modelAPIDocsResolutionInput{
					Platform: section.Platform, Kind: platform.RuntimeKind(), Protocol: platform.Protocol, Model: model.Name,
					Capabilities: platform.Capabilities, BillingModes: billingModes,
				})
				targets = append(targets, modelPlazaCatalogTarget{
					channelIndex: channelIndex, sectionIndex: sectionIndex, modelIndex: modelIndex, input: input,
				})
				lookupTargets = append(lookupTargets, service.ModelAPIDocsBinding{
					Platform: input.Platform, Protocol: input.Protocol, Model: input.Model,
				})
			}
		}
	}

	savedBindings, loadErr := h.settingService.GetXimoAIModelAPIDocsBindings(ctx, lookupTargets)
	if loadErr != nil && isAdmin {
		return fmt.Errorf("load model API documentation configuration: %w", loadErr)
	}
	for index, target := range targets {
		var saved *service.ModelAPIDocsBinding
		if loadErr == nil {
			saved = savedBindings[index]
			if saved != nil {
				if validationErr := validateXimoAIModelAPIDocsBinding(*saved, target.input.Capabilities, target.input.Kind); validationErr != nil {
					if isAdmin {
						return fmt.Errorf("stored model API documentation configuration is invalid: %w", validationErr)
					}
					saved = nil
				}
			}
		}
		details := buildModelPlazaModelDetails(target.input, saved, isAdmin)
		model := &channels[target.channelIndex].Platforms[target.sectionIndex].SupportedModels[target.modelIndex]
		model.Types = details.Types
		model.Capabilities = details.Capabilities
		model.Protocols = details.Protocols
		model.InvocationModes = details.InvocationModes
		model.APIDocumentation = &details.APIDocumentation
	}
	return h.enrichXimoAIModelBrands(ctx, channels, isAdmin)
}

func buildModelPlazaModelDetails(
	input modelAPIDocsResolutionInput,
	saved *service.ModelAPIDocsBinding,
	isAdmin bool,
) modelPlazaModelDetails {
	automatic := resolveXimoAIModelAPIDocsAutomaticBinding(input)
	effective := automatic
	source := "automatic"
	if saved != nil {
		effective = *saved
		source = "administrator"
	}
	profiles := applyModelAPIDocsTemplateValues(selectedXimoAIModelAPIDocsProfiles(effective), input.Model)
	documentation := modelAPIDocsResponse{
		Platform: input.Platform, Protocol: input.Protocol, Model: input.Model, Source: source,
		Binding: effective, Profiles: profiles,
	}
	if isAdmin {
		documentation.Editor = &modelAPIDocsEditorResponse{
			AutomaticBinding:  automatic,
			AvailableProfiles: applyModelAPIDocsTemplateValues(compatibleModelAPIDocsProfiles(input), input.Model),
		}
	}

	types := make([]string, 0, len(effective.Categories))
	for _, category := range effective.Categories {
		types = appendUniqueModelCatalogValue(types, category.Category)
	}
	capabilities := make([]string, 0)
	protocols := make([]string, 0)
	modes := make([]string, 0)
	for _, profile := range profiles {
		capability := profile.capability
		if capability == "" {
			switch profile.Category {
			case modelDocsCategoryImage:
				capability = service.PlatformCapabilityImages
			case modelDocsCategoryVideo:
				capability = service.PlatformCapabilityVideos
			case modelDocsCategoryTTS, modelDocsCategoryASR:
				capability = service.PlatformCapabilityAudio
			}
		}
		capabilities = appendUniqueModelCatalogValue(capabilities, capability)
		protocols = appendUniqueModelCatalogValue(protocols, profile.Protocol)
		for _, variant := range profile.Variants {
			modes = appendUniqueModelCatalogValue(modes, variant.Mode)
		}
	}
	return modelPlazaModelDetails{
		Types: types, Capabilities: capabilities, Protocols: protocols,
		InvocationModes: modes, APIDocumentation: documentation,
	}
}

func appendUniqueModelCatalogValue(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}
