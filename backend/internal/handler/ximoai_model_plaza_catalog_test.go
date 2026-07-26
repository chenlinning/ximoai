package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildModelPlazaMetadataReturnsCompactPublicContract(t *testing.T) {
	details := buildModelPlazaMetadata(modelMetadataResolutionInput{
		Platform:     "custom-openai",
		Kind:         "",
		Protocol:     service.PlatformProtocolOpenAICompatible,
		Model:        "custom-model",
		Capabilities: []string{service.PlatformCapabilityResponses},
		BillingModes: []string{string(service.BillingModeToken)},
	}, nil, false)

	require.Equal(t, "Other", details.Brand)
	require.Equal(t, []string{modelTypeConversation}, details.Types)
	require.Equal(t, []string{"sync", "stream"}, details.InvocationModes)
	require.Nil(t, details.Editor)
}

func TestAutomaticMetadataUsesActualUpstreamModelForCustomPlatform(t *testing.T) {
	details := buildModelPlazaMetadata(modelMetadataResolutionInput{
		Platform:         "custom-openai-compatible",
		Protocol:         service.PlatformProtocolOpenAICompatible,
		Model:            "ximo-gpt",
		RecognitionModel: "gpt-5",
	}, nil, false)

	require.Equal(t, "OpenAI", details.Brand)
	require.Equal(t, []string{modelTypeConversation}, details.Types)
	require.ElementsMatch(t, []string{modelInvocationSync, modelInvocationStream}, details.InvocationModes)
	require.Equal(t, []string{"none", "low", "medium", "high", "xhigh", "max"}, details.ReasoningLevels)
	require.False(t, details.ThinkingSupported)
}

func TestModelPlazaKeepsRecognitionNameOutOfPublicModelJSON(t *testing.T) {
	model := userSupportedModel{
		Name:            "ximo-gpt",
		Platform:        "custom-openai-compatible",
		recognitionName: "gpt-5",
	}
	raw, err := json.Marshal(model)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "gpt-5")
	require.NotContains(t, string(raw), "recognition")
}

func TestModelPlazaModelJSONContainsOnlyPublicCatalogFields(t *testing.T) {
	details := buildModelPlazaMetadata(modelMetadataResolutionInput{
		Platform: "custom-openai", Protocol: service.PlatformProtocolOpenAICompatible, Model: "custom-model",
		Capabilities: []string{service.PlatformCapabilityResponses}, BillingModes: []string{string(service.BillingModeToken)},
	}, nil, false)
	model := userSupportedModel{
		Name: "custom-model", Platform: "custom-openai", Brand: details.Brand, Pricing: &userSupportedModelPricing{BillingMode: string(service.BillingModeToken)},
		Types: details.Types, InvocationModes: details.InvocationModes,
	}
	raw, err := json.Marshal(model)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, key := range []string{"name", "platform", "brand", "pricing", "types", "invocation_modes"} {
		_, exists := decoded[key]
		require.Truef(t, exists, "model catalog must expose %q", key)
	}
	for _, key := range []string{"metadata_editor", "brand_editor", "capabilities", "protocols", "api_documentation"} {
		require.NotContains(t, decoded, key)
	}
	serialized := strings.ToLower(string(raw))
	for _, forbidden := range []string{"api_key", "base_url", "upstream_url", "mapped_model", "account_id"} {
		require.NotContains(t, serialized, forbidden)
	}
}

func TestAvailableChannelModelJSONOmitsUnsetModelPlazaBrand(t *testing.T) {
	raw, err := json.Marshal(userSupportedModel{Name: "custom-model", Platform: "custom-openai"})
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"brand"`)
	require.NotContains(t, string(raw), `"brand_editor"`)
}

func TestBuildModelPlazaMetadataUsesAdministratorOverridesAndCompleteOptions(t *testing.T) {
	brand := "XimoAI Lab"
	types := []string{modelTypeTTS}
	modes := []string{modelInvocationSync, modelInvocationBidirectional}
	levels := []string{"low", "high"}
	thinking := true
	saved := &service.ModelMetadataOverride{
		Platform: "custom-openai", Model: "image-model", Brand: &brand, Types: &types, InvocationModes: &modes,
		ReasoningLevels: &levels, ThinkingSupported: &thinking,
	}
	details := buildModelPlazaMetadata(modelMetadataResolutionInput{
		Platform: saved.Platform, Protocol: service.PlatformProtocolOpenAICompatible, Model: saved.Model,
		Capabilities: []string{service.PlatformCapabilityImages},
		BillingModes: []string{string(service.BillingModeImage)},
	}, saved, true)

	require.Equal(t, brand, details.Brand)
	require.Equal(t, types, details.Types)
	require.Equal(t, modes, details.InvocationModes)
	require.Equal(t, levels, details.ReasoningLevels)
	require.True(t, details.ThinkingSupported)
	require.NotNil(t, details.Editor)
	require.Equal(t, allModelTypeOptions(), details.Editor.Options.Types)
	require.Equal(t, allModelInvocationModeOptions(), details.Editor.Options.InvocationModes)
	require.Equal(t, service.XimoAIModelReasoningLevelOptions(), details.Editor.Options.ReasoningLevels)
}

func TestAutomaticVolcengineSpeechMetadataCoversEveryVoiceMode(t *testing.T) {
	details := buildModelPlazaMetadata(modelMetadataResolutionInput{
		Platform: service.PlatformVolcengineAgentPlan,
		Kind:     service.PlatformKindVolcengineAgentPlan,
		Protocol: service.PlatformProtocolNative,
		Model:    service.VolcengineAgentPlanTTSModel,
	}, nil, false)

	require.Equal(t, []string{modelTypeTTS}, details.Types)
	require.ElementsMatch(t, []string{modelInvocationSync, modelInvocationStream, modelInvocationBidirectional}, details.InvocationModes)
}

func TestAutomaticVolcengineASRMetadataMatchesWebSocketRoutes(t *testing.T) {
	details := resolveAutomaticModelMetadata(modelMetadataResolutionInput{
		Platform:         service.PlatformVolcengineAgentPlan,
		Kind:             service.PlatformKindVolcengineAgentPlan,
		Model:            service.VolcengineAgentPlanASRModel,
		RecognitionModel: service.VolcengineAgentPlanASRModel,
	})

	require.Equal(t, []string{modelTypeASR}, details.Types)
	require.Equal(t, []string{modelInvocationStream}, details.InvocationModes)
}

func TestAutomaticVolcengineSeedreamMetadataIncludesStreaming(t *testing.T) {
	details := resolveAutomaticModelMetadata(modelMetadataResolutionInput{
		Platform:         service.PlatformVolcengineAgentPlan,
		Kind:             service.PlatformKindVolcengineAgentPlan,
		Model:            service.VolcengineAgentPlanSeedreamModel,
		RecognitionModel: service.VolcengineAgentPlanSeedreamModel,
	})

	require.Equal(t, []string{modelTypeImage}, details.Types)
	require.ElementsMatch(t, []string{modelInvocationSync, modelInvocationStream}, details.InvocationModes)
}
