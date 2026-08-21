package service

type officialTokenPriceCard struct {
	Model                 string
	Provider              string
	Input                 float64
	Output                float64
	CacheRead             float64
	CacheWrite            float64
	InputPriority         float64
	OutputPriority        float64
	CacheReadPriority     float64
	CacheWritePriority    float64
	LongContextThreshold  int
	LongContextInputMult  float64
	LongContextOutputMult float64
	SupportsPromptCaching bool
	SupportsServiceTier   bool
}

func officialTokenPriceCards() []officialTokenPriceCard {
	return []officialTokenPriceCard{
		{Model: "kimi-k3", Provider: "moonshot", Input: 3e-6, Output: 15e-6, CacheRead: 0.30e-6, SupportsPromptCaching: true},
		{Model: "kimi-k2.7-code", Provider: "moonshot", Input: 0.95e-6, Output: 4e-6, CacheRead: 0.19e-6, SupportsPromptCaching: true},
		{Model: "kimi-k2.7-code-highspeed", Provider: "moonshot", Input: 1.90e-6, Output: 8e-6, CacheRead: 0.38e-6, SupportsPromptCaching: true},
		{Model: "kimi-k2.6", Provider: "moonshot", Input: 0.95e-6, Output: 4e-6, CacheRead: 0.16e-6, SupportsPromptCaching: true},
		{Model: "kimi-k2.5", Provider: "moonshot", Input: 0.60e-6, Output: 3e-6, CacheRead: 0.10e-6, SupportsPromptCaching: true},
		{Model: "moonshot-v1-8k", Provider: "moonshot", Input: 0.20e-6, Output: 2e-6},
		{Model: "moonshot-v1-32k", Provider: "moonshot", Input: 1e-6, Output: 3e-6},
		{Model: "moonshot-v1-128k", Provider: "moonshot", Input: 2e-6, Output: 5e-6},
		{Model: "moonshot-v1-8k-vision-preview", Provider: "moonshot", Input: 0.20e-6, Output: 2e-6},
		{Model: "moonshot-v1-32k-vision-preview", Provider: "moonshot", Input: 1e-6, Output: 3e-6},
		{Model: "moonshot-v1-128k-vision-preview", Provider: "moonshot", Input: 2e-6, Output: 5e-6},

		{Model: "minimax-m3", Provider: "minimax", Input: 0.30e-6, Output: 1.20e-6, CacheRead: 0.06e-6, InputPriority: 0.45e-6, OutputPriority: 1.80e-6, CacheReadPriority: 0.09e-6, LongContextThreshold: 512000, LongContextInputMult: 2, LongContextOutputMult: 2, SupportsPromptCaching: true, SupportsServiceTier: true},
		{Model: "minimax-m2.7", Provider: "minimax", Input: 0.30e-6, Output: 1.20e-6, CacheRead: 0.06e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},
		{Model: "minimax-m2.7-highspeed", Provider: "minimax", Input: 0.60e-6, Output: 2.40e-6, CacheRead: 0.06e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},
		{Model: "minimax-m2.5", Provider: "minimax", Input: 0.30e-6, Output: 1.20e-6, CacheRead: 0.03e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},
		{Model: "minimax-m2.5-highspeed", Provider: "minimax", Input: 0.60e-6, Output: 2.40e-6, CacheRead: 0.03e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},
		{Model: "minimax-m2.1", Provider: "minimax", Input: 0.30e-6, Output: 1.20e-6, CacheRead: 0.03e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},
		{Model: "minimax-m2.1-highspeed", Provider: "minimax", Input: 0.60e-6, Output: 2.40e-6, CacheRead: 0.03e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},
		{Model: "minimax-m2", Provider: "minimax", Input: 0.30e-6, Output: 1.20e-6, CacheRead: 0.03e-6, CacheWrite: 0.375e-6, SupportsPromptCaching: true},

		{Model: "deepseek-v4-flash", Provider: "deepseek", Input: 0.44e-6, Output: 1.32e-6, CacheRead: 0.014e-6, SupportsPromptCaching: true},
		{Model: "deepseek-v4-pro", Provider: "deepseek", Input: 1.32e-6, Output: 3.96e-6, CacheRead: 0.044e-6, SupportsPromptCaching: true},
		{Model: "deepseek-v4-flash-vision-exp", Provider: "deepseek", Input: 0.44e-6, Output: 1.32e-6, CacheRead: 0.014e-6, SupportsPromptCaching: true},

		{Model: "glm-5.3", Provider: "zhipu", Input: 1.4e-6, Output: 4.4e-6, CacheRead: 0.26e-6, SupportsPromptCaching: true},
		{Model: "glm-5.2", Provider: "zhipu", Input: 1.4e-6, Output: 4.4e-6, CacheRead: 0.26e-6, SupportsPromptCaching: true},
		{Model: "glm-5.1", Provider: "zhipu", Input: 1.4e-6, Output: 4.4e-6, CacheRead: 0.26e-6, SupportsPromptCaching: true},
		{Model: "glm-5", Provider: "zhipu", Input: 1e-6, Output: 3.2e-6, CacheRead: 0.2e-6, SupportsPromptCaching: true},
		{Model: "glm-5-turbo", Provider: "zhipu", Input: 1.2e-6, Output: 4e-6, CacheRead: 0.24e-6, SupportsPromptCaching: true},
		{Model: "glm-4.7", Provider: "zhipu", Input: 0.6e-6, Output: 2.2e-6, CacheRead: 0.11e-6, SupportsPromptCaching: true},
		{Model: "glm-4.7-flashx", Provider: "zhipu", Input: 0.07e-6, Output: 0.4e-6, CacheRead: 0.01e-6, SupportsPromptCaching: true},
		{Model: "glm-4.6", Provider: "zhipu", Input: 0.6e-6, Output: 2.2e-6, CacheRead: 0.11e-6, SupportsPromptCaching: true},
		{Model: "glm-4.5", Provider: "zhipu", Input: 0.6e-6, Output: 2.2e-6, CacheRead: 0.11e-6, SupportsPromptCaching: true},
		{Model: "glm-4.5-x", Provider: "zhipu", Input: 2.2e-6, Output: 8.9e-6, CacheRead: 0.45e-6, SupportsPromptCaching: true},
	}
}

func (c officialTokenPriceCard) liteLLMPricing() *LiteLLMModelPricing {
	return &LiteLLMModelPricing{
		InputCostPerToken:                   c.Input,
		OutputCostPerToken:                  c.Output,
		CacheReadInputTokenCost:             c.CacheRead,
		CacheCreationInputTokenCost:         c.CacheWrite,
		InputCostPerTokenPriority:           c.InputPriority,
		OutputCostPerTokenPriority:          c.OutputPriority,
		CacheReadInputTokenCostPriority:     c.CacheReadPriority,
		CacheCreationInputTokenCostPriority: c.CacheWritePriority,
		LongContextInputTokenThreshold:      c.LongContextThreshold,
		LongContextInputCostMultiplier:      c.LongContextInputMult,
		LongContextOutputCostMultiplier:     c.LongContextOutputMult,
		SupportsPromptCaching:               c.SupportsPromptCaching,
		SupportsServiceTier:                 c.SupportsServiceTier,
		LiteLLMProvider:                     c.Provider,
		Mode:                                "chat",
	}
}

func (c officialTokenPriceCard) modelPricing() *ModelPricing {
	return &ModelPricing{
		InputPricePerToken:                 c.Input,
		OutputPricePerToken:                c.Output,
		CacheReadPricePerToken:             c.CacheRead,
		CacheCreationPricePerToken:         c.CacheWrite,
		InputPricePerTokenPriority:         c.InputPriority,
		OutputPricePerTokenPriority:        c.OutputPriority,
		CacheReadPricePerTokenPriority:     c.CacheReadPriority,
		CacheCreationPricePerTokenPriority: c.CacheWritePriority,
		LongContextInputThreshold:          c.LongContextThreshold,
		LongContextInputMultiplier:         c.LongContextInputMult,
		LongContextOutputMultiplier:        c.LongContextOutputMult,
		SupportsCacheBreakdown:             false,
	}
}

func currentOfficialLiteLLMPricingOverrides() map[string]*LiteLLMModelPricing {
	out := make(map[string]*LiteLLMModelPricing)
	for _, card := range officialTokenPriceCards() {
		out[card.Model] = card.liteLLMPricing()
	}
	return out
}

func currentOfficialBillingFallbacks() map[string]*ModelPricing {
	out := make(map[string]*ModelPricing)
	for _, card := range officialTokenPriceCards() {
		out[card.Model] = card.modelPricing()
	}
	return out
}
