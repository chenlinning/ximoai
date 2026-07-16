package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
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

func TestGrokCompactSchedulingPreservesBuiltinError(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	groupID := int64(92001)
	accounts := []Account{{
		ID:          72001,
		Platform:    PlatformGrok,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, err := svc.selectAccountWithLoadAwareness(
		context.Background(),
		&groupID,
		PlatformGrok,
		"",
		"",
		nil,
		true,
		"",
		true,
	)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoAvailableCompactAccounts), "err=%v", err)
	require.Nil(t, selection)
}
