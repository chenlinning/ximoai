package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestXimoAIPricingOverrideMatchesBillingFallbackCards(t *testing.T) {
	path := filepath.Join("..", "..", "resources", "model-pricing", "ximoai_official_pricing_overrides.json")
	body, err := os.ReadFile(path)
	require.NoError(t, err)

	svc := &PricingService{cfg: &config.Config{}}
	catalog, err := svc.parsePricingData(body)
	require.NoError(t, err)

	cards := officialTokenPriceCards()
	require.Len(t, catalog, len(cards))
	for _, card := range cards {
		actual := catalog[card.Model]
		require.NotNil(t, actual, card.Model)
		require.InDelta(t, card.Input, actual.InputCostPerToken, 1e-12, card.Model)
		require.InDelta(t, card.Output, actual.OutputCostPerToken, 1e-12, card.Model)
		require.InDelta(t, card.CacheRead, actual.CacheReadInputTokenCost, 1e-12, card.Model)
		require.InDelta(t, card.CacheWrite, actual.CacheCreationInputTokenCost, 1e-12, card.Model)
		require.InDelta(t, card.InputPriority, actual.InputCostPerTokenPriority, 1e-12, card.Model)
		require.InDelta(t, card.OutputPriority, actual.OutputCostPerTokenPriority, 1e-12, card.Model)
		require.InDelta(t, card.CacheReadPriority, actual.CacheReadInputTokenCostPriority, 1e-12, card.Model)
		require.InDelta(t, card.CacheWritePriority, actual.CacheCreationInputTokenCostPriority, 1e-12, card.Model)
		require.Equal(t, card.LongContextThreshold, actual.LongContextInputTokenThreshold, card.Model)
		require.InDelta(t, card.LongContextInputMult, actual.LongContextInputCostMultiplier, 1e-12, card.Model)
		require.InDelta(t, card.LongContextOutputMult, actual.LongContextOutputCostMultiplier, 1e-12, card.Model)
		require.Equal(t, card.Provider, actual.LiteLLMProvider, card.Model)
		require.Equal(t, card.SupportsPromptCaching, actual.SupportsPromptCaching, card.Model)
		require.Equal(t, card.SupportsServiceTier, actual.SupportsServiceTier, card.Model)
	}
}
