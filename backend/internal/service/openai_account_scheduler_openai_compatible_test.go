package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayService_SelectAccountWithScheduler_EnabledSelectsOpenAICompatiblePlatform(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10108)
	accounts := []Account{
		{
			ID:          37101,
			Platform:    "acme",
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.acme.test"},
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          37102,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		"acme",
		&groupID,
		"resp_ignored_for_custom_platform",
		"session_custom_platform",
		"gpt-image-2",
		nil,
		OpenAIUpstreamTransportAny,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37101), selection.Account.ID)
	require.Equal(t, "acme", selection.Account.Platform)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}
