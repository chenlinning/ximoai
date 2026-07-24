package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestXimoAIGrokVideoRuntimeKindSurvivesPlatformRename(t *testing.T) {
	account := &Account{
		Platform: "video-provider-renamed",
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			PlatformKindCredentialKey: PlatformKindGrokVideo,
		},
	}

	require.True(t, account.IsGrokVideo())
	eligible, reason := account.GrokMediaGenerationEligibility()
	require.True(t, eligible)
	require.Equal(t, "grok_video_api_key", reason)
	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityGrokMediaGeneration))
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions))
}

func TestXimoAIAudioRuntimeKindsSurvivePlatformRename(t *testing.T) {
	openAIAudio := &Account{
		Platform: "audio-provider-renamed",
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			PlatformKindCredentialKey: PlatformKindOpenAIAudio,
		},
	}
	klingAudio := &Account{
		Platform: "kling-provider-renamed",
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			PlatformKindCredentialKey: PlatformKindKlingAudio,
		},
	}

	require.True(t, openAIAudio.IsOpenAIAudio())
	require.True(t, klingAudio.IsKlingAudio())
	require.False(t, openAIAudio.IsKlingAudio())
	require.False(t, klingAudio.IsOpenAIAudio())
}

func TestXimoAIBuiltinPlatformsRequireExpectedBaseURLs(t *testing.T) {
	require.True(t, (Platform{Kind: PlatformKindGrokVideo}).RequiresAPIKeyBaseURL())
	require.True(t, (Platform{Kind: PlatformKindKlingAudio}).RequiresAPIKeyBaseURL())
	require.False(t, (Platform{Kind: PlatformKindOpenAIAudio}).RequiresAPIKeyBaseURL())
}

func TestBuiltinSchedulerPlatformsIncludeAllXimoAIMediaPlatforms(t *testing.T) {
	platforms := builtinSchedulerPlatforms()
	require.Contains(t, platforms, PlatformGrokVideo)
	require.Contains(t, platforms, PlatformOpenAIAudio)
	require.Contains(t, platforms, PlatformKlingAudio)
}

func TestXimoAIGrokVideoRuntimeKindIsSelectedBySchedulerAfterRename(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	const platform = "video-provider-renamed"
	groupID := int64(9016)
	account := Account{
		ID:          916,
		Platform:    platform,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{PlatformKindCredentialKey: PlatformKindGrokVideo},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, _, err := svc.SelectAccountWithSchedulerForCapability(
		context.Background(), &groupID, "", "", "grok-imagine-video", nil,
		OpenAIUpstreamTransportHTTPSSE, OpenAIEndpointCapabilityGrokMediaGeneration,
		false, false, false, platform,
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
}
