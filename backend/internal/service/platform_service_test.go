package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type platformRepoStubForPlatformService struct {
	platforms map[string]Platform
	usage     PlatformUsage
	deleted   string
}

func (s *platformRepoStubForPlatformService) List(_ context.Context, includeDisabled bool) ([]Platform, error) {
	out := make([]Platform, 0, len(s.platforms))
	for _, platform := range s.platforms {
		if includeDisabled || platform.Enabled {
			out = append(out, platform)
		}
	}
	return out, nil
}

func (s *platformRepoStubForPlatformService) GetBySlug(_ context.Context, slug string) (*Platform, error) {
	if platform, ok := s.platforms[NormalizePlatformSlug(slug)]; ok {
		cp := platform
		return &cp, nil
	}
	return nil, ErrPlatformNotFound
}

func (s *platformRepoStubForPlatformService) Create(_ context.Context, platform *Platform) error {
	if s.platforms == nil {
		s.platforms = map[string]Platform{}
	}
	s.platforms[platform.Slug] = *platform
	return nil
}

func (s *platformRepoStubForPlatformService) Update(_ context.Context, platform *Platform) error {
	if s.platforms == nil {
		return ErrPlatformNotFound
	}
	s.platforms[platform.Slug] = *platform
	return nil
}

func (s *platformRepoStubForPlatformService) Delete(_ context.Context, slug string) error {
	s.deleted = NormalizePlatformSlug(slug)
	delete(s.platforms, s.deleted)
	return nil
}

func (s *platformRepoStubForPlatformService) Usage(_ context.Context, _ string) (PlatformUsage, error) {
	return s.usage, nil
}

func TestPlatformService_DeleteRejectsPlatformInUse(t *testing.T) {
	repo := &platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			"acme": {
				Slug:        "acme",
				DisplayName: "Acme",
				Protocol:    PlatformProtocolOpenAICompatible,
				BaseURL:     "https://api.acme.test",
				AuthModes:   []string{AccountTypeAPIKey},
				Enabled:     true,
			},
		},
		usage: PlatformUsage{AccountCount: 1},
	}
	svc := NewPlatformService(repo)

	err := svc.Delete(context.Background(), "acme")
	require.Error(t, err)
	require.Contains(t, err.Error(), "PLATFORM_IN_USE")
	require.Empty(t, repo.deleted)
}

func TestPlatformService_DeleteAllowsUnusedCustomPlatform(t *testing.T) {
	repo := &platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			"acme": {
				Slug:        "acme",
				DisplayName: "Acme",
				Protocol:    PlatformProtocolOpenAICompatible,
				BaseURL:     "https://api.acme.test",
				AuthModes:   []string{AccountTypeAPIKey},
				Enabled:     true,
			},
		},
	}
	svc := NewPlatformService(repo)

	require.NoError(t, svc.Delete(context.Background(), "acme"))
	require.Equal(t, "acme", repo.deleted)
}

func TestPlatformService_CreateRejectsCustomNativeProtocol(t *testing.T) {
	repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{}}
	svc := NewPlatformService(repo)

	_, err := svc.Create(context.Background(), Platform{
		Slug:         "custom-native",
		DisplayName:  "Custom Native",
		Protocol:     PlatformProtocolNative,
		AuthModes:    []string{AccountTypeAPIKey},
		Capabilities: []string{"messages"},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_CUSTOM_PLATFORM_PROTOCOL")
	require.Empty(t, repo.platforms)
}

func TestPlatformService_CreateNormalizesOpenAICompatiblePlatform(t *testing.T) {
	repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{}}
	svc := NewPlatformService(repo)

	created, err := svc.Create(context.Background(), Platform{
		Slug:        " Acme ",
		DisplayName: "Acme",
		Protocol:    PlatformProtocolOpenAICompatible,
		BaseURL:     "https://api.acme.test/v1/",
		AuthModes:   []string{AccountTypeOAuth, AccountTypeAPIKey},
		Color:       "#123456",
	})

	require.NoError(t, err)
	require.Equal(t, "acme", created.Slug)
	require.Equal(t, "https://api.acme.test/v1", created.BaseURL)
	require.Equal(t, []string{AccountTypeAPIKey}, created.AuthModes)
	require.ElementsMatch(t, []string{
		PlatformCapabilityResponses,
		PlatformCapabilityChatCompletions,
		PlatformCapabilityImages,
		PlatformCapabilityVideos,
		PlatformCapabilityAudio,
		PlatformCapabilityRealtime,
	}, created.Capabilities)
	require.True(t, created.Enabled)
	require.False(t, created.Builtin)
}

func TestPlatformService_BuiltinPlatformsExposeDefaultBaseURLs(t *testing.T) {
	platforms := builtinPlatforms(true)

	baseURLs := map[string]string{}
	for _, platform := range platforms {
		baseURLs[platform.Slug] = platform.BaseURL
	}

	require.Equal(t, PlatformDefaultBaseURLAnthropic, baseURLs[PlatformAnthropic])
	require.Equal(t, PlatformDefaultBaseURLOpenAI, baseURLs[PlatformOpenAI])
	require.Equal(t, PlatformDefaultBaseURLGemini, baseURLs[PlatformGemini])
	require.Equal(t, PlatformDefaultBaseURLAntigravity, baseURLs[PlatformAntigravity])
}

func TestPlatformService_CreateAssignsDeterministicColor(t *testing.T) {
	repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{}}
	svc := NewPlatformService(repo)

	first, err := svc.Create(context.Background(), Platform{
		Slug:        "alpha-api",
		DisplayName: "Alpha API",
		Protocol:    PlatformProtocolOpenAICompatible,
		BaseURL:     "https://alpha.test/v1",
		AuthModes:   []string{AccountTypeAPIKey},
	})
	require.NoError(t, err)

	second, err := svc.Create(context.Background(), Platform{
		Slug:        "alpha-api-2",
		DisplayName: "Alpha API 2",
		Protocol:    PlatformProtocolOpenAICompatible,
		BaseURL:     "https://alpha2.test/v1",
		AuthModes:   []string{AccountTypeAPIKey},
	})
	require.NoError(t, err)

	require.NotEmpty(t, first.Color)
	require.NotEmpty(t, second.Color)
	require.NotEqual(t, first.Color, second.Color)
}

func TestPlatformService_CreateNormalizesCustomProtocolPlatforms(t *testing.T) {
	tests := []struct {
		name         string
		protocol     string
		capabilities []string
	}{
		{
			name:         "anthropic",
			protocol:     PlatformProtocolAnthropic,
			capabilities: []string{PlatformCapabilityMessages},
		},
		{
			name:         "gemini",
			protocol:     PlatformProtocolGemini,
			capabilities: []string{PlatformCapabilityMessages, PlatformCapabilityNativeGemini},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{}}
			svc := NewPlatformService(repo)

			created, err := svc.Create(context.Background(), Platform{
				Slug:        "custom-" + tt.name,
				DisplayName: "Custom " + tt.name,
				Protocol:    tt.protocol,
				BaseURL:     "https://api." + tt.name + ".test/",
				AuthModes:   []string{AccountTypeOAuth, AccountTypeAPIKey},
			})

			require.NoError(t, err)
			require.Equal(t, strings.TrimRight("https://api."+tt.name+".test/", "/"), created.BaseURL)
			require.Equal(t, []string{AccountTypeAPIKey}, created.AuthModes)
			require.ElementsMatch(t, tt.capabilities, created.Capabilities)
			require.False(t, created.Builtin)
		})
	}
}

func TestPlatformService_ValidateAccountPlatformCustomPlatformsAreAPIKeyOnly(t *testing.T) {
	repo := &platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			"custom-gemini": {
				Slug:      "custom-gemini",
				Protocol:  PlatformProtocolGemini,
				AuthModes: []string{AccountTypeAPIKey},
				Enabled:   true,
				Builtin:   false,
			},
		},
	}
	svc := NewPlatformService(repo)

	require.NoError(t, svc.ValidateAccountPlatform(context.Background(), "custom-gemini", AccountTypeAPIKey))
	err := svc.ValidateAccountPlatform(context.Background(), "custom-gemini", AccountTypeOAuth)
	require.Error(t, err)
	require.Contains(t, err.Error(), "UNSUPPORTED_PLATFORM_AUTH_MODE")
}
