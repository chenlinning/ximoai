package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXimoAIGrokVideoRuntimeKindUsesFixedPlatform(t *testing.T) {
	account := &Account{
		Platform: PlatformGrokVideo,
		Type:     AccountTypeAPIKey,
	}

	require.True(t, account.IsGrokVideo())
	eligible, reason := account.GrokMediaGenerationEligibility()
	require.True(t, eligible)
	require.Equal(t, "grok_video_api_key", reason)
	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityGrokMediaGeneration))
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
}

func TestXimoAIAudioRuntimeKindsUseFixedPlatforms(t *testing.T) {
	openAIAudio := &Account{
		Platform: PlatformOpenAIAudio,
		Type:     AccountTypeAPIKey,
	}
	klingAudio := &Account{
		Platform: PlatformKlingAudio,
		Type:     AccountTypeAPIKey,
	}

	require.True(t, openAIAudio.IsOpenAIAudio())
	require.True(t, klingAudio.IsKlingAudio())
	require.False(t, openAIAudio.IsKlingAudio())
	require.False(t, klingAudio.IsOpenAIAudio())
}

func TestXimoAIBuiltinPlatformsRequireExpectedBaseURLs(t *testing.T) {
	require.True(t, XimoAIPlatformRequiresAPIKeyBaseURL(PlatformGrokVideo))
	require.True(t, XimoAIPlatformRequiresAPIKeyBaseURL(PlatformKlingAudio))
	require.False(t, XimoAIPlatformRequiresAPIKeyBaseURL(PlatformOpenAIAudio))
	require.False(t, XimoAIPlatformRequiresAPIKeyBaseURL(PlatformVolcengineAgentPlan))
}

func TestXimoAIVolcengineAgentPlanRuntimeKind(t *testing.T) {
	account := &Account{
		Platform: PlatformVolcengineAgentPlan,
		Type:     AccountTypeAPIKey,
	}

	require.Equal(t, PlatformKindVolcengineAgentPlan, account.PlatformRuntimeKind())
	require.True(t, IsXimoAIMediaPlatformKind(account.PlatformRuntimeKind()))
}

func TestVolcengineAgentPlanDefaultModelsList(t *testing.T) {
	require.Equal(t, []string{
		VolcengineAgentPlanSeedreamModel,
		VolcengineAgentPlanTTSModel,
		VolcengineAgentPlanASRModel,
	}, defaultModelsListCandidateIDs(PlatformVolcengineAgentPlan))
}

func TestVolcengineAgentPlanDefaultsImageGenerationEnabled(t *testing.T) {
	require.True(t, defaultAllowImageGenerationForPlatform(PlatformVolcengineAgentPlan))
}

func TestVolcengineAgentPlanIsAFixedSchedulerPlatform(t *testing.T) {
	platforms := schedulerSnapshotPlatforms()
	require.Contains(t, platforms, PlatformVolcengineAgentPlan)
}
