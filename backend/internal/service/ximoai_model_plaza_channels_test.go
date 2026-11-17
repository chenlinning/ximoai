//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestListAvailableChannelPricing_KeepsOnlyChannelPricedModels(t *testing.T) {
	inputPrice := 0.000001
	zeroPrice := 0.0
	channels := []Channel{{
		ID:       1,
		Name:     "priced-channel",
		Status:   StatusActive,
		GroupIDs: []int64{1},
		ModelMapping: map[string]map[string]string{
			"openai": {
				"gpt-priced":   "gpt-priced",
				"gpt-zero":     "gpt-zero",
				"gpt-unpriced": "gpt-unpriced",
			},
		},
		ModelPricing: []ChannelModelPricing{
			{
				Platform:   "openai",
				Models:     []string{"gpt-priced"},
				InputPrice: &inputPrice,
			},
			{
				Platform:   "openai",
				Models:     []string{"gpt-zero"},
				InputPrice: &zeroPrice,
			},
		},
	}}
	groupRepo := &stubGroupRepoForAvailable{
		activeGroups: []Group{{ID: 1, Name: "default", Platform: "openai"}},
	}
	svc := newAvailableChannelService(channels, groupRepo)

	out, err := svc.ListAvailableChannelPricing(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].SupportedModels, 2)
	require.Equal(t, "gpt-priced", out[0].SupportedModels[0].Name)
	require.Equal(t, "gpt-zero", out[0].SupportedModels[1].Name)
	require.NotNil(t, out[0].SupportedModels[0].Pricing)
	require.NotNil(t, out[0].SupportedModels[1].Pricing)
}

func TestListAvailableChannelPricing_HidesMappedTargetsAndKeepsAliasPricing(t *testing.T) {
	proPrice := 0.8
	flashPrice := 0.00006
	channels := []Channel{{
		ID:       1,
		Name:     "gemini-channel",
		Status:   StatusActive,
		GroupIDs: []int64{1},
		ModelMapping: map[string]map[string]string{
			"gemini": {
				"NanoBananaPro": "gemini-3-pro-image",
				"NanoBanana2":   "gemini-3.1-flash-image",
			},
		},
		ModelPricing: []ChannelModelPricing{
			{
				Platform:        "gemini",
				Models:          []string{"gemini-3-pro-image"},
				BillingMode:     BillingModeImage,
				PerRequestPrice: &proPrice,
			},
			{
				Platform:        "gemini",
				Models:          []string{"gemini-3.1-flash-image"},
				BillingMode:     BillingModeImage,
				PerRequestPrice: &flashPrice,
			},
		},
	}}
	groupRepo := &stubGroupRepoForAvailable{
		activeGroups: []Group{{ID: 1, Name: "gemini", Platform: "gemini"}},
	}
	svc := newAvailableChannelService(channels, groupRepo)

	out, err := svc.ListAvailableChannelPricing(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].SupportedModels, 2)

	byName := make(map[string]SupportedModel, len(out[0].SupportedModels))
	for _, model := range out[0].SupportedModels {
		byName[model.Name] = model
	}
	require.Contains(t, byName, "NanoBananaPro")
	require.Contains(t, byName, "NanoBanana2")
	require.NotContains(t, byName, "gemini-3-pro-image")
	require.NotContains(t, byName, "gemini-3.1-flash-image")
	require.NotNil(t, byName["NanoBananaPro"].Pricing)
	require.NotNil(t, byName["NanoBananaPro"].Pricing.PerRequestPrice)
	require.InDelta(t, proPrice, *byName["NanoBananaPro"].Pricing.PerRequestPrice, 1e-12)
	require.NotNil(t, byName["NanoBanana2"].Pricing)
	require.NotNil(t, byName["NanoBanana2"].Pricing.PerRequestPrice)
	require.InDelta(t, flashPrice, *byName["NanoBanana2"].Pricing.PerRequestPrice, 1e-12)
}
