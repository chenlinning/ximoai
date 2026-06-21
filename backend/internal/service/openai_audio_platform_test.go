package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIAudioCustomAPIKeySupportsChatOnly(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAIAudio,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"platform_protocol": PlatformProtocolOpenAICompatible,
		},
	}

	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityEmbeddings))
}

func TestBuiltinSchedulerPlatformsIncludesOpenAIAudio(t *testing.T) {
	require.Contains(t, builtinSchedulerPlatforms(), PlatformOpenAIAudio)
}

func TestOpenAIAudioAlwaysUsesRawChatCompletions(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]any
	}{
		{
			name:  "without responses extra",
			extra: nil,
		},
		{
			name: "force responses cannot override platform route",
			extra: map[string]any{
				"openai_responses_mode": "force_responses",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformOpenAIAudio,
				Type:     AccountTypeAPIKey,
				Extra:    tt.extra,
			}

			require.True(t, shouldUseRawChatCompletionsForAccount(account))
		})
	}
}

func TestOpenAIAudioRouteDoesNotChangeOfficialOpenAIUnknownDefault(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}

	require.False(t, shouldUseRawChatCompletionsForAccount(account))
}
