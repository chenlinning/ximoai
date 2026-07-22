package service

import (
	"context"
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

func TestWeightedStickyFallbackAllowsCustomOpenAICompatibleCompactRequest(t *testing.T) {
	account := Account{
		ID:          72002,
		Platform:    "acme-openai",
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"platform_protocol": PlatformProtocolOpenAICompatible,
		},
	}
	service := &OpenAIGatewayService{
		accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
	}
	scheduler := &defaultOpenAIAccountScheduler{service: service}

	selection, err := scheduler.tryFallbackToWeightedSticky(context.Background(), OpenAIAccountScheduleRequest{
		Platform:        account.Platform,
		StickyWeighted:  true,
		StickyAccountID: account.ID,
		RequireCompact:  true,
	})

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
}
