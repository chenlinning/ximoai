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
			require.NotEmpty(t, variant.Termination)
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
			for _, workflowStep := range variant.Steps {
				require.Contains(t, workflowStep.Parameters, ModelAPIDocsParameter{
					Name: "Authorization", Location: "header", Required: true, Type: "string",
					Description: "Bearer API key for the XimoAI gateway.",
				})
			}
		}
	}
}

func TestXimoAIModelAPIDocsOffersCompleteOpenAICompatibleEditorProfiles(t *testing.T) {
	profiles := compatibleModelAPIDocsProfiles(modelAPIDocsResolutionInput{
		Platform: "custom-openai", Protocol: service.PlatformProtocolOpenAICompatible, Model: "custom-model",
		Capabilities: []string{
			service.PlatformCapabilityResponses,
			service.PlatformCapabilityChatCompletions,
			service.PlatformCapabilityEmbeddings,
			service.PlatformCapabilityImages,
			service.PlatformCapabilityVideos,
			service.PlatformCapabilityAudio,
			service.PlatformCapabilityRealtime,
		},
	})
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ID)
	}

	for _, expected := range []string{
		"openai_responses", "openai_responses_realtime", "openai_chat_completions",
		"openai_embeddings", "openai_image_generation",
	} {
		require.Contains(t, ids, expected)
	}
	require.NotContains(t, ids, "openai_video_generation")
	require.NotContains(t, ids, "openai_audio_speech")
	require.NotContains(t, ids, "openai_live_call")
}

func TestXimoAIModelAPIDocsRecognizesRenamedCustomizedBuiltinKinds(t *testing.T) {
	tests := []struct {
		name         string
		kind         string
		capabilities []string
		profiles     []string
	}{
		{
			name: "grok video", kind: service.PlatformKindGrokVideo,
			capabilities: []string{service.PlatformCapabilityVideos},
			profiles:     []string{"ximoai_grok_video_generation", "ximoai_grok_video_extension"},
		},
		{
			name: "openai audio", kind: service.PlatformKindOpenAIAudio,
			capabilities: []string{service.PlatformCapabilityChatCompletions, service.PlatformCapabilityAudio},
			profiles:     []string{"openai_chat_completions", "openai_audio_speech", "openai_audio_transcription", "openai_audio_translation"},
		},
		{
			name: "kling audio", kind: service.PlatformKindKlingAudio,
			capabilities: []string{service.PlatformCapabilityAudio},
			profiles:     []string{"openai_audio_speech"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiles := compatibleModelAPIDocsProfiles(modelAPIDocsResolutionInput{
				Platform: "renamed-platform", Kind: tt.kind, Protocol: service.PlatformProtocolGemini,
				Model: "custom-model", Capabilities: tt.capabilities,
			})
			ids := make([]string, 0, len(profiles))
			for _, profile := range profiles {
				ids = append(ids, profile.ID)
			}
			require.ElementsMatch(t, tt.profiles, ids)
		})
	}
}

func TestXimoAIModelAPIDocsGrokVideoDoesNotAdvertiseUnsupportedSteps(t *testing.T) {
	profiles := compatibleModelAPIDocsProfiles(modelAPIDocsResolutionInput{
		Platform: "renamed-video", Kind: service.PlatformKindGrokVideo,
		Protocol: service.PlatformProtocolOpenAICompatible, Model: "grok-video-3",
		Capabilities: []string{service.PlatformCapabilityVideos},
	})

	for _, profile := range profiles {
		require.NotEqual(t, "openai_video_edit", profile.ID)
		for _, variant := range profile.Variants {
			for _, workflowStep := range variant.Steps {
				require.NotContains(t, workflowStep.Path, "/content")
			}
		}
		if profile.ID == "ximoai_grok_video_extension" {
			require.Contains(t, profile.Variants[0].Steps[0].RequestExample, `"video"`)
		}
	}
}

func TestXimoAIModelAPIDocsOffersLiveOnlyForOfficialOpenAI(t *testing.T) {
	profiles := compatibleModelAPIDocsProfiles(modelAPIDocsResolutionInput{
		Platform: service.PlatformOpenAI, Protocol: service.PlatformProtocolOpenAI, Model: "gpt-live",
		Capabilities: []string{service.PlatformCapabilityRealtime},
	})
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ID)
	}

	require.Contains(t, ids, "openai_live_call")
	require.Contains(t, ids, "openai_live_sideband")
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
	requests := map[string]string{}
	for _, profile := range profiles {
		if len(profile.Variants) > 0 && len(profile.Variants[0].Steps) > 0 {
			paths[profile.ID] = profile.Variants[0].Steps[0].Path
			requests[profile.ID] = profile.Variants[0].Steps[0].RequestExample
		}
	}
	require.Equal(t, "/v1/images/generations/async", paths["openai_image_generation_async"])
	require.Equal(t, "/v1/images/edits/async", paths["openai_image_edit_async"])
	require.Contains(t, requests["openai_image_edit"], `"image"`)
	require.Contains(t, requests["openai_image_edit_async"], `"image"`)
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
				Platform:     "renamed-volcengine",
				Kind:         service.PlatformKindVolcengineAgentPlan,
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
	available := []string{}
	for _, profile := range compatibleModelAPIDocsProfiles(input) {
		available = append(available, profile.ID)
	}
	require.Contains(t, available, "openai_responses_realtime")
	require.NotContains(t, modelAPIDocsBindingProfileIDs(resolveXimoAIModelAPIDocsAutomaticBinding(input)), "openai_responses_realtime")
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
	}, ""))
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

	err := validateXimoAIModelAPIDocsBinding(binding, []string{service.PlatformCapabilityMessages}, "")
	require.Error(t, err)
}
