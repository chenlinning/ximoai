package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type ximoAIModelsAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (s *ximoAIModelsAccountRepoStub) ListSchedulableByGroupID(_ context.Context, _ int64) ([]Account, error) {
	return append([]Account(nil), s.accounts...), nil
}

type ximoAIModelsChannelRepoStub struct {
	ChannelRepository
	channel        Channel
	groupPlatforms map[int64]string
}

func (s *ximoAIModelsChannelRepoStub) ListAll(_ context.Context) ([]Channel, error) {
	return []Channel{s.channel}, nil
}

func (s *ximoAIModelsChannelRepoStub) GetGroupPlatforms(_ context.Context, _ []int64) (map[int64]string, error) {
	return s.groupPlatforms, nil
}

func TestGetXimoAIAvailableModels_PrefersChannelMappingSources(t *testing.T) {
	groupID := int64(41)
	platform := "acme"
	accountRepo := &ximoAIModelsAccountRepoStub{accounts: []Account{{
		ID:       1,
		Platform: platform,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"account-model": "upstream-model",
			},
		},
	}}}
	channelRepo := &ximoAIModelsChannelRepoStub{
		channel: Channel{
			ID:       9,
			Status:   StatusActive,
			GroupIDs: []int64{groupID},
			ModelMapping: map[string]map[string]string{
				platform: {
					"channel-model-b": "account-model-b",
					"channel-model-a": "account-model-a",
				},
			},
		},
		groupPlatforms: map[int64]string{groupID: platform},
	}
	svc := &GatewayService{
		accountRepo:    accountRepo,
		channelService: NewChannelService(channelRepo, nil, nil, nil),
	}

	models := svc.GetXimoAIAvailableModels(context.Background(), &groupID, platform)

	require.Equal(t, []string{"channel-model-a", "channel-model-b"}, models)
}

func TestGetXimoAIAvailableModels_FallsBackWithoutPlatformMapping(t *testing.T) {
	groupID := int64(42)
	platform := "acme"
	accountRepo := &ximoAIModelsAccountRepoStub{accounts: []Account{{
		ID:       1,
		Platform: platform,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"account-model": "upstream-model",
			},
		},
	}}}
	channelRepo := &ximoAIModelsChannelRepoStub{
		channel: Channel{
			ID:       10,
			Status:   StatusActive,
			GroupIDs: []int64{groupID},
			ModelMapping: map[string]map[string]string{
				"other-platform": {"other-model": "other-upstream"},
			},
		},
		groupPlatforms: map[int64]string{groupID: platform},
	}
	svc := &GatewayService{
		accountRepo:    accountRepo,
		channelService: NewChannelService(channelRepo, nil, nil, nil),
	}

	models := svc.GetXimoAIAvailableModels(context.Background(), &groupID, platform)

	require.Equal(t, []string{"account-model"}, models)
}

type ximoAIModelsPlatformRepoStub struct {
	PlatformRepository
	platform Platform
}

func (s *ximoAIModelsPlatformRepoStub) GetBySlug(_ context.Context, slug string) (*Platform, error) {
	if NormalizePlatformSlug(slug) != s.platform.Slug {
		return nil, ErrPlatformNotFound
	}
	platform := s.platform
	return &platform, nil
}

func TestXimoAIModelsFallbackPlatform_LeavesBuiltinPlatformUnchanged(t *testing.T) {
	svc := NewPlatformService(&ximoAIModelsPlatformRepoStub{platform: Platform{
		Slug:     PlatformGrokVideo,
		Protocol: PlatformProtocolOpenAICompatible,
		Enabled:  true,
		Builtin:  true,
	}})

	require.Equal(t, PlatformGrokVideo, svc.XimoAIModelsFallbackPlatform(context.Background(), PlatformGrokVideo))
}
