package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/ximoai/modelcatalog"
)

const (
	modelCapabilityResponses       = "responses"
	modelCapabilityChatCompletions = "chat_completions"
	modelCapabilityEmbeddings      = "embeddings"
	modelCapabilityImages          = "images"
	modelCapabilityVideos          = "videos"
	modelCapabilityAudio           = "audio"
	modelCapabilityMessages        = "messages"
	modelCapabilityNativeGemini    = "native_gemini"
)

const (
	modelTypeConversation = service.ModelTypeConversation
	modelTypeEmbedding    = service.ModelTypeEmbedding
	modelTypeImage        = service.ModelTypeImage
	modelTypeVideo        = service.ModelTypeVideo
	modelTypeTTS          = service.ModelTypeTTS
	modelTypeASR          = service.ModelTypeASR

	modelInvocationSync          = service.ModelInvocationSync
	modelInvocationStream        = service.ModelInvocationStream
	modelInvocationAsync         = service.ModelInvocationAsync
	modelInvocationBidirectional = service.ModelInvocationBidirectional
	modelInvocationBatch         = service.ModelInvocationBatch
)

type modelMetadataResolutionInput struct {
	Platform         string
	Kind             string
	Protocol         string
	Model            string
	RecognitionModel string
	Capabilities     []string
	BillingModes     []string
}

type modelMetadataValues struct {
	Brand             string   `json:"brand"`
	Types             []string `json:"types"`
	InvocationModes   []string `json:"invocation_modes"`
	ReasoningLevels   []string `json:"reasoning_levels,omitempty"`
	ThinkingSupported bool     `json:"thinking_supported,omitempty"`
}

type modelMetadataOverrideResponse struct {
	Brand             *string   `json:"brand,omitempty"`
	Types             *[]string `json:"types,omitempty"`
	InvocationModes   *[]string `json:"invocation_modes,omitempty"`
	ReasoningLevels   *[]string `json:"reasoning_levels,omitempty"`
	ThinkingSupported *bool     `json:"thinking_supported,omitempty"`
}

type modelMetadataOptionsResponse struct {
	Types           []string `json:"types"`
	InvocationModes []string `json:"invocation_modes"`
	ReasoningLevels []string `json:"reasoning_levels"`
}

type modelMetadataEditorResponse struct {
	Automatic modelMetadataValues            `json:"automatic"`
	Override  *modelMetadataOverrideResponse `json:"override,omitempty"`
	Options   modelMetadataOptionsResponse   `json:"options"`
}

type modelMetadataDetails struct {
	modelMetadataValues
	Editor *modelMetadataEditorResponse `json:"editor,omitempty"`
}

func allModelTypeOptions() []string {
	return service.XimoAIModelTypeOptions()
}

func allModelInvocationModeOptions() []string {
	return service.XimoAIModelInvocationModeOptions()
}

func buildModelPlazaMetadata(
	input modelMetadataResolutionInput,
	override *service.ModelMetadataOverride,
	isAdmin bool,
) modelMetadataDetails {
	automatic := resolveAutomaticModelMetadata(input)
	effective := automatic
	if override != nil {
		if override.Brand != nil {
			effective.Brand = *override.Brand
		}
		if override.Types != nil {
			effective.Types = append([]string(nil), (*override.Types)...)
		}
		if override.InvocationModes != nil {
			effective.InvocationModes = append([]string(nil), (*override.InvocationModes)...)
		}
		if override.ReasoningLevels != nil {
			effective.ReasoningLevels = append([]string(nil), (*override.ReasoningLevels)...)
		}
		if override.ThinkingSupported != nil {
			effective.ThinkingSupported = *override.ThinkingSupported
		}
	}
	details := modelMetadataDetails{modelMetadataValues: effective}
	if isAdmin {
		details.Editor = &modelMetadataEditorResponse{
			Automatic: automatic,
			Options: modelMetadataOptionsResponse{
				Types:           allModelTypeOptions(),
				InvocationModes: allModelInvocationModeOptions(),
				ReasoningLevels: service.XimoAIModelReasoningLevelOptions(),
			},
		}
		if override != nil {
			details.Editor.Override = &modelMetadataOverrideResponse{
				Brand:             cloneStringPointer(override.Brand),
				Types:             cloneStringSlicePointer(override.Types),
				InvocationModes:   cloneStringSlicePointer(override.InvocationModes),
				ReasoningLevels:   cloneStringSlicePointer(override.ReasoningLevels),
				ThinkingSupported: cloneBoolPointer(override.ThinkingSupported),
			}
		}
	}
	return details
}

func resolveAutomaticModelMetadata(input modelMetadataResolutionInput) modelMetadataValues {
	input = normalizeModelMetadataResolutionInput(input)
	metadata := modelMetadataValues{Brand: detectXimoAIModelBrand(input.RecognitionModel), Types: []string{}, InvocationModes: []string{}}
	if registered, ok := modelcatalog.PublicMetadataFor(input.Platform, input.RecognitionModel); ok {
		return modelMetadataValues{
			Brand:             registered.Brand,
			Types:             append([]string(nil), registered.Types...),
			InvocationModes:   append([]string(nil), registered.InvocationModes...),
			ReasoningLevels:   append([]string(nil), registered.ReasoningLevels...),
			ThinkingSupported: registered.ThinkingSupported,
		}
	}

	for _, billingMode := range input.BillingModes {
		switch billingMode {
		case string(service.BillingModeToken):
			if containsModelMetadataValue(input.Capabilities, modelCapabilityEmbeddings) &&
				!containsAnyModelMetadataValue(input.Capabilities, modelCapabilityResponses, modelCapabilityChatCompletions, modelCapabilityMessages, modelCapabilityNativeGemini) {
				appendModelMetadata(&metadata, modelTypeEmbedding, modelInvocationSync)
			} else {
				appendModelMetadata(&metadata, modelTypeConversation, modelInvocationSync, modelInvocationStream)
			}
		case string(service.BillingModeImage):
			appendModelMetadata(&metadata, modelTypeImage, modelInvocationSync, modelInvocationAsync)
		case string(service.BillingModeVideo):
			appendModelMetadata(&metadata, modelTypeVideo, modelInvocationAsync)
		case string(service.BillingModePerRequest):
			resolvePerRequestModelMetadata(&metadata, input.Capabilities)
		}
	}
	if len(metadata.Types) == 0 {
		resolveCapabilityOnlyModelMetadata(&metadata, input.Capabilities)
	}
	return metadata
}

func resolvePerRequestModelMetadata(metadata *modelMetadataValues, capabilities []string) {
	mediaCapabilities := make([]string, 0, 3)
	for _, capability := range []string{
		modelCapabilityImages,
		modelCapabilityVideos,
		modelCapabilityAudio,
	} {
		if containsModelMetadataValue(capabilities, capability) {
			mediaCapabilities = append(mediaCapabilities, capability)
		}
	}
	if len(mediaCapabilities) != 1 {
		return
	}
	switch mediaCapabilities[0] {
	case modelCapabilityVideos:
		appendModelMetadata(metadata, modelTypeVideo, modelInvocationAsync)
	case modelCapabilityImages:
		appendModelMetadata(metadata, modelTypeImage, modelInvocationSync, modelInvocationAsync)
	case modelCapabilityAudio:
		appendModelMetadata(metadata, modelTypeTTS, modelInvocationSync)
	}
}

func resolveCapabilityOnlyModelMetadata(metadata *modelMetadataValues, capabilities []string) {
	if containsAnyModelMetadataValue(capabilities, modelCapabilityResponses, modelCapabilityChatCompletions, modelCapabilityMessages, modelCapabilityNativeGemini) {
		appendModelMetadata(metadata, modelTypeConversation, modelInvocationSync, modelInvocationStream)
	} else if containsModelMetadataValue(capabilities, modelCapabilityEmbeddings) {
		appendModelMetadata(metadata, modelTypeEmbedding, modelInvocationSync)
	}
}

func appendModelMetadata(metadata *modelMetadataValues, modelType string, modes ...string) {
	metadata.Types = appendUniqueModelMetadataValue(metadata.Types, modelType)
	for _, mode := range modes {
		metadata.InvocationModes = appendUniqueModelMetadataValue(metadata.InvocationModes, mode)
	}
}

func appendUniqueModelMetadataValue(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func normalizeModelMetadataResolutionInput(input modelMetadataResolutionInput) modelMetadataResolutionInput {
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Protocol = strings.ToLower(strings.TrimSpace(input.Protocol))
	input.Model = strings.TrimSpace(input.Model)
	input.RecognitionModel = strings.TrimSpace(input.RecognitionModel)
	if input.RecognitionModel == "" {
		input.RecognitionModel = input.Model
	}
	input.Capabilities = normalizeModelMetadataValues(input.Capabilities)
	input.BillingModes = normalizeModelMetadataValues(input.BillingModes)
	return input
}

func normalizeModelMetadataValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out = appendUniqueModelMetadataValue(out, value)
		}
	}
	return out
}

func containsModelMetadataValue(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func containsAnyModelMetadataValue(values []string, targets ...string) bool {
	for _, target := range targets {
		if containsModelMetadataValue(values, target) {
			return true
		}
	}
	return false
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneStringSlicePointer(value *[]string) *[]string {
	if value == nil {
		return nil
	}
	copy := append([]string(nil), (*value)...)
	return &copy
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
