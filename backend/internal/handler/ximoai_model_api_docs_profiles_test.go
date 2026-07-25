package handler

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestXimoAIModelAPIDocsProfilesAreStructurallyValid(t *testing.T) {
	profiles := XimoAIModelAPIDocsProfiles()
	require.NotEmpty(t, profiles)
	validCategories := []string{
		modelDocsCategoryConversation,
		modelDocsCategoryImage,
		modelDocsCategoryVideo,
		modelDocsCategoryTTS,
		modelDocsCategoryASR,
	}
	profileIDs := map[string]struct{}{}
	for _, profile := range profiles {
		require.NotEmpty(t, profile.ID)
		require.Contains(t, validCategories, profile.Category)
		require.NotEmpty(t, profile.Protocol)
		require.NotEmpty(t, profile.Variants)
		_, duplicate := profileIDs[profile.ID]
		require.False(t, duplicate, "duplicate profile id %s", profile.ID)
		profileIDs[profile.ID] = struct{}{}

		variantIDs := map[string]struct{}{}
		for _, variant := range profile.Variants {
			require.Contains(t, []string{"sync", "stream", "async", "bidirectional"}, variant.Mode)
			require.Contains(t, []string{"http", "websocket"}, variant.Transport)
			require.Contains(t, []string{"json", "sse", "binary", "websocket_frames"}, variant.Delivery)
			require.NotEmpty(t, variant.Steps)
			_, duplicate = variantIDs[variant.ID]
			require.False(t, duplicate, "duplicate variant id %s/%s", profile.ID, variant.ID)
			variantIDs[variant.ID] = struct{}{}
			if variant.Delivery == "sse" {
				require.Equal(t, "http", variant.Transport)
			}
			if variant.Transport == "websocket" {
				require.Equal(t, "websocket_frames", variant.Delivery)
				require.Equal(t, "GET", variant.Steps[0].Method)
			}
		}
	}
}

func TestXimoAIModelAPIDocsVolcengineProfileMatrix(t *testing.T) {
	type expectedProfile struct {
		category  string
		method    string
		path      string
		mode      string
		transport string
		delivery  string
	}
	expected := map[string]expectedProfile{
		"volcengine_images_generations":        {modelDocsCategoryImage, "POST", "/v1/volcengine/images/generations", "sync", "http", "json"},
		"volcengine_tts_unidirectional":        {modelDocsCategoryTTS, "POST", "/v1/volcengine/audio/tts/unidirectional", "sync", "http", "binary"},
		"volcengine_tts_unidirectional_stream": {modelDocsCategoryTTS, "GET", "/v1/volcengine/audio/tts/unidirectional/stream", "stream", "websocket", "websocket_frames"},
		"volcengine_tts_bidirectional":         {modelDocsCategoryTTS, "GET", "/v1/volcengine/audio/tts/bidirection", "bidirectional", "websocket", "websocket_frames"},
		"volcengine_asr_bigmodel_async":        {modelDocsCategoryASR, "GET", "/v1/volcengine/audio/asr/bigmodel_async", "async", "websocket", "websocket_frames"},
		"volcengine_asr_bigmodel_nostream":     {modelDocsCategoryASR, "GET", "/v1/volcengine/audio/asr/bigmodel_nostream", "sync", "websocket", "websocket_frames"},
	}

	actual := make(map[string]expectedProfile)
	for _, profile := range XimoAIModelAPIDocsProfiles() {
		if !strings.HasPrefix(profile.ID, "volcengine_") {
			continue
		}
		require.Len(t, profile.Variants, 1, profile.ID)
		require.Len(t, profile.Variants[0].Steps, 1, profile.ID)
		variant := profile.Variants[0]
		step := variant.Steps[0]
		actual[profile.ID] = expectedProfile{profile.Category, step.Method, step.Path, variant.Mode, variant.Transport, variant.Delivery}
	}
	require.Equal(t, expected, actual)
}

func TestXimoAIModelAPIDocsProfilesDoNotExposeUpstreamEndpointsOrCredentials(t *testing.T) {
	for _, profile := range XimoAIModelAPIDocsProfiles() {
		raw := strings.ToLower(profile.Description)
		for _, variant := range profile.Variants {
			for _, step := range variant.Steps {
				raw += strings.ToLower(step.Path + step.RequestExample + step.ResponseExample)
			}
		}
		require.NotContains(t, raw, "api.openai.com")
		require.NotContains(t, raw, "openspeech.bytedance.com")
		require.NotContains(t, raw, "api.x.ai")
		require.NotContains(t, raw, "sk-")
	}
}

func TestXimoAIModelAPIDocsAsyncImageProfilesKeepDistinctSubmitPaths(t *testing.T) {
	profiles := XimoAIModelAPIDocsProfiles()
	paths := map[string]string{}
	for _, profile := range profiles {
		if len(profile.Variants) > 0 && len(profile.Variants[0].Steps) > 0 {
			paths[profile.ID] = profile.Variants[0].Steps[0].Path
		}
	}
	require.Equal(t, "/v1/images/generations/async", paths["openai_image_generation_async"])
	require.Equal(t, "/v1/images/edits/async", paths["openai_image_edit_async"])
}

func TestXimoAIModelAPIDocsAutomaticVolcengineBindings(t *testing.T) {
	tests := []struct {
		model    string
		profiles []string
	}{
		{service.VolcengineAgentPlanSeedreamModel, []string{"volcengine_images_generations"}},
		{service.VolcengineAgentPlanTTSModel, []string{"volcengine_tts_unidirectional", "volcengine_tts_unidirectional_stream", "volcengine_tts_bidirectional"}},
		{service.VolcengineAgentPlanASRModel, []string{"volcengine_asr_bigmodel_async", "volcengine_asr_bigmodel_nostream"}},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			binding := resolveXimoAIModelAPIDocsAutomaticBinding(modelAPIDocsResolutionInput{
				Platform:     service.PlatformVolcengineAgentPlan,
				Protocol:     service.PlatformProtocolNative,
				Model:        tt.model,
				Capabilities: []string{service.PlatformCapabilityImages, service.PlatformCapabilityAudio},
			})
			require.ElementsMatch(t, tt.profiles, modelAPIDocsBindingProfileIDs(binding))
		})
	}
}

func TestXimoAIModelAPIDocsAutomaticOpenAIBindingKeepsSyncAndStream(t *testing.T) {
	binding := resolveXimoAIModelAPIDocsAutomaticBinding(modelAPIDocsResolutionInput{
		Platform:     "custom-openai",
		Protocol:     service.PlatformProtocolOpenAICompatible,
		Model:        "custom-model",
		Capabilities: []string{service.PlatformCapabilityResponses, service.PlatformCapabilityChatCompletions},
		BillingModes: []string{string(service.BillingModeToken)},
	})

	require.ElementsMatch(t, []string{"openai_responses", "openai_chat_completions"}, modelAPIDocsBindingProfileIDs(binding))
	for _, category := range binding.Categories {
		for _, endpoint := range category.Endpoints {
			require.ElementsMatch(t, []string{"sync", "stream"}, endpoint.Variants)
		}
	}
}

func TestXimoAIModelAPIDocsAutomaticAnthropicAndGeminiBindingsKeepSyncAndStream(t *testing.T) {
	tests := []struct {
		name       string
		platform   string
		protocol   string
		capability string
		profile    string
	}{
		{"anthropic", "custom-anthropic", service.PlatformProtocolAnthropic, service.PlatformCapabilityMessages, "anthropic_messages"},
		{"gemini", "custom-gemini", service.PlatformProtocolGemini, service.PlatformCapabilityNativeGemini, "gemini_content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := resolveXimoAIModelAPIDocsAutomaticBinding(modelAPIDocsResolutionInput{
				Platform: tt.platform, Protocol: tt.protocol, Model: "custom-model",
				Capabilities: []string{tt.capability}, BillingModes: []string{string(service.BillingModeToken)},
			})
			require.Equal(t, []string{tt.profile}, modelAPIDocsBindingProfileIDs(binding))
			require.ElementsMatch(t, []string{"sync", "stream"}, binding.Categories[0].Endpoints[0].Variants)
		})
	}
}

func TestXimoAIModelAPIDocsRealtimeRequiresExplicitCapability(t *testing.T) {
	input := modelAPIDocsResolutionInput{
		Platform: "custom-openai", Protocol: service.PlatformProtocolOpenAICompatible, Model: "realtime-model",
		Capabilities: []string{service.PlatformCapabilityResponses}, BillingModes: []string{string(service.BillingModeToken)},
	}
	require.NotContains(t, modelAPIDocsBindingProfileIDs(resolveXimoAIModelAPIDocsAutomaticBinding(input)), "openai_responses_realtime")

	input.Capabilities = append(input.Capabilities, service.PlatformCapabilityRealtime)
	require.Contains(t, modelAPIDocsBindingProfileIDs(resolveXimoAIModelAPIDocsAutomaticBinding(input)), "openai_responses_realtime")
}

func TestXimoAIModelAPIDocsBindingAllowsMultipleCategories(t *testing.T) {
	binding := service.ModelAPIDocsBinding{
		Platform: "custom-openai", Protocol: service.PlatformProtocolOpenAICompatible, Model: "multimodal-model",
		Categories: []service.ModelAPIDocsCategoryBinding{
			{Category: modelDocsCategoryConversation, Endpoints: []service.ModelAPIDocsEndpointBinding{{Profile: "openai_responses", Variants: []string{"sync", "stream"}}}},
			{Category: modelDocsCategoryImage, Endpoints: []service.ModelAPIDocsEndpointBinding{{Profile: "openai_image_generation", Variants: []string{"sync"}}}},
		},
	}
	require.NoError(t, validateXimoAIModelAPIDocsBinding(binding, []string{
		service.PlatformCapabilityResponses,
		service.PlatformCapabilityImages,
	}))
	require.Len(t, selectedXimoAIModelAPIDocsProfiles(binding), 2)
}

func TestXimoAIModelAPIDocsPerRequestDoesNotGuessCategory(t *testing.T) {
	binding := resolveXimoAIModelAPIDocsAutomaticBinding(modelAPIDocsResolutionInput{
		Platform:     "custom-openai",
		Protocol:     service.PlatformProtocolOpenAICompatible,
		Model:        "ambiguous-model",
		Capabilities: []string{service.PlatformCapabilityChatCompletions, service.PlatformCapabilityAudio},
		BillingModes: []string{string(service.BillingModePerRequest)},
	})

	require.Empty(t, binding.Categories)
}

func TestXimoAIModelAPIDocsProtocolChangeDoesNotReuseIncompatibleProfiles(t *testing.T) {
	binding := service.ModelAPIDocsBinding{
		Platform: "custom",
		Protocol: service.PlatformProtocolAnthropic,
		Model:    "model",
		Categories: []service.ModelAPIDocsCategoryBinding{{
			Category: "conversation",
			Endpoints: []service.ModelAPIDocsEndpointBinding{{
				Profile:  "openai_responses",
				Variants: []string{"sync"},
			}},
		}},
	}

	err := validateXimoAIModelAPIDocsBinding(binding, []string{service.PlatformCapabilityMessages})
	require.Error(t, err)
}
