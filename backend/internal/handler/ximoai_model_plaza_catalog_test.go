package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildModelPlazaModelDetailsReturnsCompletePublicContract(t *testing.T) {
	details := buildModelPlazaModelDetails(modelAPIDocsResolutionInput{
		Platform:     "custom-openai",
		Protocol:     service.PlatformProtocolOpenAICompatible,
		Model:        "custom-model",
		Capabilities: []string{service.PlatformCapabilityResponses},
		BillingModes: []string{string(service.BillingModeToken)},
	}, nil, false)

	require.Equal(t, []string{modelDocsCategoryConversation}, details.Types)
	require.Equal(t, []string{service.PlatformCapabilityResponses}, details.Capabilities)
	require.Equal(t, []string{"openai_responses"}, details.Protocols)
	require.Equal(t, []string{"sync", "stream"}, details.InvocationModes)
	require.Equal(t, "automatic", details.APIDocumentation.Source)
	require.Len(t, details.APIDocumentation.Profiles, 1)
	require.Nil(t, details.APIDocumentation.Editor)
}

func TestModelPlazaModelJSONContainsOnlyPublicCatalogFields(t *testing.T) {
	details := buildModelPlazaModelDetails(modelAPIDocsResolutionInput{
		Platform: "custom-openai", Protocol: service.PlatformProtocolOpenAICompatible, Model: "custom-model",
		Capabilities: []string{service.PlatformCapabilityResponses}, BillingModes: []string{string(service.BillingModeToken)},
	}, nil, false)
	model := userSupportedModel{
		Name: "custom-model", Platform: "custom-openai", Pricing: &userSupportedModelPricing{BillingMode: string(service.BillingModeToken)},
		Types: details.Types, Capabilities: details.Capabilities, Protocols: details.Protocols,
		InvocationModes: details.InvocationModes, APIDocumentation: &details.APIDocumentation,
	}
	raw, err := json.Marshal(model)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, key := range []string{"name", "platform", "pricing", "types", "capabilities", "protocols", "invocation_modes", "api_documentation"} {
		_, exists := decoded[key]
		require.Truef(t, exists, "model catalog must expose %q", key)
	}
	serialized := strings.ToLower(string(raw))
	for _, forbidden := range []string{"api_key", "base_url", "upstream_url", "mapped_model", "account_id"} {
		require.NotContains(t, serialized, forbidden)
	}
}

func TestBuildModelPlazaModelDetailsUsesAdministratorBindingAndEditor(t *testing.T) {
	saved := &service.ModelAPIDocsBinding{
		Platform: "custom-openai", Protocol: service.PlatformProtocolOpenAICompatible, Model: "image-model",
		Categories: []service.ModelAPIDocsCategoryBinding{{
			Category: modelDocsCategoryImage,
			Endpoints: []service.ModelAPIDocsEndpointBinding{{
				Profile: "openai_image_generation", Variants: []string{"sync"},
			}},
		}},
	}
	details := buildModelPlazaModelDetails(modelAPIDocsResolutionInput{
		Platform: saved.Platform, Protocol: saved.Protocol, Model: saved.Model,
		Capabilities: []string{service.PlatformCapabilityImages},
		BillingModes: []string{string(service.BillingModeImage)},
	}, saved, true)

	require.Equal(t, []string{modelDocsCategoryImage}, details.Types)
	require.Equal(t, []string{service.PlatformCapabilityImages}, details.Capabilities)
	require.Equal(t, []string{"openai_images"}, details.Protocols)
	require.Equal(t, []string{"sync"}, details.InvocationModes)
	require.Equal(t, "administrator", details.APIDocumentation.Source)
	require.NotNil(t, details.APIDocumentation.Editor)
}
