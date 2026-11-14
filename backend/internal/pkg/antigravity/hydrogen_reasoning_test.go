package antigravity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHydrogenReasoningGenerationConfig(t *testing.T) {
	temperature, topP, topK := 0.3, 0.7, 20
	for _, model := range []string{"gemini-4-hydrogen", "gemini-4-hydrogen-high", "gemini-4-hydrogen-xhigh"} {
		t.Run(model, func(t *testing.T) {
			require.True(t, IsGeminiReasoningModel(model))
			cfg := buildGenerationConfig(&ClaudeRequest{
				Model: model, MaxTokens: 64000, Temperature: &temperature, TopP: &topP, TopK: &topK,
				Thinking: &ThinkingConfig{Type: "enabled", BudgetTokens: 1024},
			})
			require.Equal(t, 64000, cfg.MaxOutputTokens)
			require.Nil(t, cfg.StopSequences)
			require.Nil(t, cfg.Temperature)
			require.Nil(t, cfg.TopP)
			require.Nil(t, cfg.TopK)
			require.NotNil(t, cfg.ThinkingConfig)
			require.Equal(t, 1024, cfg.ThinkingConfig.ThinkingBudget)
			encoded, err := json.Marshal(cfg)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "stopSequences")
			require.Contains(t, string(encoded), `thinkingBudget`)
		})
	}
	require.False(t, IsGeminiReasoningModel("gemini-4-hydrogenation"))
	legacy := buildGenerationConfig(&ClaudeRequest{Model: "gemini-2.5-flash", Temperature: &temperature, TopP: &topP, TopK: &topK})
	require.NotEmpty(t, legacy.StopSequences)
	require.NotNil(t, legacy.Temperature)
	require.NotNil(t, legacy.TopP)
	require.NotNil(t, legacy.TopK)
}
