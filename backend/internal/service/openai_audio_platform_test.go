package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIAudioCustomAPIKeySupportsChatOnly(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAIAudio,
		Type:     AccountTypeAPIKey,
	}

	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityEmbeddings))
}

func TestSchedulerPlatformsIncludeXimoAIBuiltins(t *testing.T) {
	platforms := schedulerSnapshotPlatforms()
	require.Contains(t, platforms, PlatformGrokVideo)
	require.Contains(t, platforms, PlatformOpenAIAudio)
	require.Contains(t, platforms, PlatformKlingAudio)
	require.Contains(t, platforms, PlatformVolcengineAgentPlan)
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
