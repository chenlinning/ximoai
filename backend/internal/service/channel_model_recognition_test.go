package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedModelsKeepsDisplayNameAndUpstreamRecognitionNameSeparate(t *testing.T) {
	channel := &Channel{
		ModelMapping: map[string]map[string]string{
			"openai": {"ximo-gpt": "gpt-5"},
		},
		ModelPricing: []ChannelModelPricing{
			{Platform: "openai", Models: []string{"gpt-5"}, BillingMode: BillingModeToken},
		},
	}

	models := channel.SupportedModels()
	require.Len(t, models, 1)
	require.Equal(t, "ximo-gpt", models[0].Name)
	require.Equal(t, "gpt-5", models[0].RecognitionName)
}

func TestSupportedModelsUsesDisplayNameWhenNoMappingExists(t *testing.T) {
	channel := &Channel{
		ModelPricing: []ChannelModelPricing{
			{Platform: "openai", Models: []string{"gpt-5"}, BillingMode: BillingModeToken},
		},
	}

	models := channel.SupportedModels()
	require.Len(t, models, 1)
	require.Equal(t, models[0].Name, models[0].RecognitionName)
}
