package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayServiceRecordUsage_PreservesExplicitFinalUpstreamModel(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID:     "gateway_looped_mapping_models",
			Usage:         ClaudeUsage{InputTokens: 10, OutputTokens: 6},
			Model:         "gpt-5.6-sol",
			UpstreamModel: "gpt-5.6-sol",
			Duration:      time.Second,
		},
		APIKey:  &APIKey{ID: 501, Quota: 100},
		User:    &User{ID: 601},
		Account: &Account{ID: 701},
		ChannelUsageFields: ChannelUsageFields{
			OriginalModel:      "public-alias",
			ChannelMappedModel: "gpt-5.6-sol",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, "public-alias", usageRepo.lastLog.RequestedModel)
	require.Equal(t, "gpt-5.6-sol", usageRepo.lastLog.Model)
	require.NotNil(t, usageRepo.lastLog.UpstreamModel)
	require.Equal(t, "gpt-5.6-sol", *usageRepo.lastLog.UpstreamModel)
}

func TestOpenAIGatewayServiceRecordUsage_PreservesExplicitFinalUpstreamModel(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	svc := newOpenAIRecordUsageServiceForTest(
		usageRepo,
		&openAIRecordUsageUserRepoStub{},
		&openAIRecordUsageSubRepoStub{},
		nil,
	)

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID:     "openai_looped_mapping_models",
			Model:         "gpt-5.6-sol",
			UpstreamModel: "gpt-5.6-sol",
			Usage:         OpenAIUsage{InputTokens: 20, OutputTokens: 10},
			Duration:      time.Second,
		},
		APIKey:  &APIKey{ID: 10},
		User:    &User{ID: 20},
		Account: &Account{ID: 30},
		ChannelUsageFields: ChannelUsageFields{
			OriginalModel:      "public-alias",
			ChannelMappedModel: "gpt-5.6-sol",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, "public-alias", usageRepo.lastLog.RequestedModel)
	require.Equal(t, "gpt-5.6-sol", usageRepo.lastLog.Model)
	require.NotNil(t, usageRepo.lastLog.UpstreamModel)
	require.Equal(t, "gpt-5.6-sol", *usageRepo.lastLog.UpstreamModel)
}
