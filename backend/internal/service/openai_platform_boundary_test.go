package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAICompactRequiredForAccountPreservesBuiltinBoundary(t *testing.T) {
	require.True(t, openAICompactRequiredForAccount(&Account{Platform: PlatformOpenAI}, true))
	require.True(t, openAICompactRequiredForAccount(&Account{Platform: PlatformGrok}, true))
	require.False(t, openAICompactRequiredForAccount(&Account{
		Platform: "acme-openai",
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"platform_protocol": PlatformProtocolOpenAICompatible,
		},
	}, true))
	require.False(t, openAICompactRequiredForAccount(&Account{Platform: PlatformOpenAI}, false))
}
