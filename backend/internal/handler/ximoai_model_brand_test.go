package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectXimoAIModelBrand(t *testing.T) {
	tests := []struct {
		model string
		brand string
	}{
		{model: "gpt-5.4", brand: "OpenAI"},
		{model: "anthropic/claude-sonnet-4-6", brand: "Anthropic"},
		{model: "models/gemini-3.1-pro", brand: "Google"},
		{model: "grok-4", brand: "xAI"},
		{model: "deepseek-r1-distill-qwen-32b", brand: "DeepSeek"},
		{model: "qwen3-coder", brand: "Qwen"},
		{model: "seedream-4.0", brand: "Doubao"},
		{model: "seedance-1.0-pro", brand: "Doubao"},
		{model: "doubao-seed-1.6", brand: "Doubao"},
		{model: "kimi-k2", brand: "Moonshot AI"},
		{model: "moonshot-v1-128k", brand: "Moonshot AI"},
		{model: "tts-custom-v1", brand: ximoAIModelBrandOther},
		{model: "unknown-private-model", brand: ximoAIModelBrandOther},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			require.Equal(t, tt.brand, detectXimoAIModelBrand(tt.model))
		})
	}
}
