package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type platformRepoStubForPlatformService struct {
	platforms map[string]Platform
	renamed   string
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

func (s *platformRepoStubForPlatformService) Update(_ context.Context, platform *Platform) error {
	if s.platforms == nil {
		return ErrPlatformNotFound
	}
	s.platforms[platform.Slug] = *platform
	return nil
}

func (s *platformRepoStubForPlatformService) Rename(_ context.Context, oldSlug string, platform *Platform) error {
	if s.platforms == nil {
		return ErrPlatformNotFound
	}
	oldSlug = NormalizePlatformSlug(oldSlug)
	delete(s.platforms, oldSlug)
	s.platforms[platform.Slug] = *platform
	s.renamed = oldSlug + "->" + platform.Slug
	return nil
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
	require.Equal(t, PlatformDefaultBaseURLGrok, baseURLs[PlatformGrok])
}

func TestPlatformService_BuiltinOpenAIAudioPlatform(t *testing.T) {
	platforms := builtinPlatforms(true)

	bySlug := map[string]Platform{}
	for _, platform := range platforms {
		bySlug[platform.Slug] = platform
	}

	platform := bySlug[PlatformOpenAIAudio]
	require.Equal(t, "OpenAI Audio", platform.DisplayName)
	require.Equal(t, PlatformProtocolOpenAICompatible, platform.Protocol)
	require.Equal(t, []string{AccountTypeAPIKey}, platform.AuthModes)
	require.ElementsMatch(t, []string{PlatformCapabilityChatCompletions, PlatformCapabilityAudio}, platform.Capabilities)
	require.True(t, platform.Builtin)
	require.True(t, builtinPlatformBaseURLEditable(platform))
}

func TestPlatformService_BuiltinGrokPlatformKeepsNativeOAuthBoundary(t *testing.T) {
	platforms := builtinPlatforms(true)

	bySlug := map[string]Platform{}
	for _, platform := range platforms {
		bySlug[platform.Slug] = platform
	}

	platform := bySlug[PlatformGrok]
	require.Equal(t, "Grok", platform.DisplayName)
	require.Equal(t, PlatformProtocolOpenAICompatible, platform.Protocol)
	require.Equal(t, PlatformDefaultBaseURLGrok, platform.BaseURL)
	require.ElementsMatch(t, []string{AccountTypeOAuth, AccountTypeAPIKey}, platform.AuthModes)
	require.ElementsMatch(t, []string{
		PlatformCapabilityResponses,
		PlatformCapabilityChatCompletions,
		PlatformCapabilityImages,
		PlatformCapabilityVideos,
	}, platform.Capabilities)
	require.True(t, platform.Builtin)
	require.False(t, builtinPlatformBaseURLEditable(platform))
	require.False(t, builtinPlatformSlugEditable(platform))
}

func TestPlatformService_BuiltinGrokVideoPlatform(t *testing.T) {
	platforms := builtinPlatforms(true)

	bySlug := map[string]Platform{}
	for _, platform := range platforms {
		bySlug[platform.Slug] = platform
	}

	platform := bySlug[PlatformGrokVideo]
	require.Equal(t, "Grok-video", platform.DisplayName)
	require.Equal(t, PlatformProtocolOpenAICompatible, platform.Protocol)
	require.Equal(t, []string{AccountTypeAPIKey}, platform.AuthModes)
	require.ElementsMatch(t, []string{PlatformCapabilityVideos}, platform.Capabilities)
	require.True(t, platform.Builtin)
	require.True(t, builtinPlatformSlugEditable(platform))
	require.True(t, builtinPlatformProtocolEditable(platform))
	require.True(t, builtinPlatformBaseURLEditable(platform))
}

func TestPlatformService_UpdateAllowsXimoAIBuiltinPlatformRename(t *testing.T) {
	repo := &platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			PlatformGrokVideo: {
				Slug:         PlatformGrokVideo,
				Kind:         PlatformKindGrokVideo,
				DisplayName:  "Grok-video",
				Protocol:     PlatformProtocolOpenAICompatible,
				BaseURL:      "https://video.old.test",
				AuthModes:    []string{AccountTypeAPIKey},
				Capabilities: []string{PlatformCapabilityVideos},
				Enabled:      true,
				Builtin:      true,
			},
		},
	}
	svc := NewPlatformService(repo)

	updated, err := svc.Update(context.Background(), PlatformGrokVideo, Platform{
		Slug:         "grok-video-x",
		DisplayName:  "Grok Video X",
		Protocol:     PlatformProtocolGemini,
		BaseURL:      "https://video.new.test",
		AuthModes:    []string{AccountTypeOAuth, AccountTypeAPIKey},
		Capabilities: []string{PlatformCapabilityVideos, PlatformCapabilityNativeGemini},
		Color:        "#123456",
		Enabled:      true,
	})

	require.NoError(t, err)
	require.Equal(t, "grok-video->grok-video-x", repo.renamed)
	require.Equal(t, "grok-video-x", updated.Slug)
	require.Equal(t, PlatformKindGrokVideo, updated.Kind)
	require.Equal(t, PlatformProtocolGemini, updated.Protocol)
	require.Equal(t, "https://video.new.test", updated.BaseURL)
	require.Contains(t, updated.Capabilities, PlatformCapabilityVideos)
}

func TestPlatformService_UpdateKeepsRequiredCapabilitiesForXimoAIBuiltinKinds(t *testing.T) {
	tests := []struct {
		slug         string
		kind         string
		capabilities []string
		required     []string
	}{
		{PlatformGrokVideo, PlatformKindGrokVideo, []string{PlatformCapabilityResponses}, []string{PlatformCapabilityVideos}},
		{PlatformOpenAIAudio, PlatformKindOpenAIAudio, []string{PlatformCapabilityResponses}, []string{PlatformCapabilityChatCompletions, PlatformCapabilityAudio}},
		{PlatformKlingAudio, PlatformKindKlingAudio, []string{PlatformCapabilityResponses}, []string{PlatformCapabilityAudio}},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{
				tt.slug: {
					Slug: tt.slug, Kind: tt.kind, DisplayName: tt.slug,
					Protocol: PlatformProtocolOpenAICompatible, BaseURL: "https://example.test",
					AuthModes: []string{AccountTypeAPIKey}, Capabilities: tt.required,
					Enabled: true, Builtin: true,
				},
			}}
			svc := NewPlatformService(repo)

			updated, err := svc.Update(context.Background(), tt.slug, Platform{
				Slug: tt.slug, DisplayName: tt.slug, Protocol: PlatformProtocolOpenAICompatible,
				BaseURL: "https://example.test", AuthModes: []string{AccountTypeAPIKey},
				Capabilities: tt.capabilities, Enabled: true,
			})

			require.NoError(t, err)
			for _, capability := range tt.required {
				require.Contains(t, updated.Capabilities, capability)
			}
		})
	}
}

func TestPlatformService_UpdateRejectsCoreBuiltinPlatformRename(t *testing.T) {
	repo := &platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			PlatformGrok: {
				Slug:         PlatformGrok,
				DisplayName:  "Grok",
				Protocol:     PlatformProtocolOpenAICompatible,
				BaseURL:      "https://official.grok.test",
				AuthModes:    []string{AccountTypeOAuth, AccountTypeAPIKey},
				Capabilities: []string{PlatformCapabilityResponses, PlatformCapabilityChatCompletions},
				Enabled:      true,
				Builtin:      true,
			},
		},
	}
	svc := NewPlatformService(repo)

	updated, err := svc.Update(context.Background(), PlatformGrok, Platform{
		Slug:        "grok-custom",
		DisplayName: "Grok Custom",
		Protocol:    PlatformProtocolGemini,
		BaseURL:     "https://custom.grok.test",
		AuthModes:   []string{AccountTypeAPIKey},
		Enabled:     true,
	})

	require.Error(t, err)
	require.Nil(t, updated)
	require.Empty(t, repo.renamed)
}

func TestPlatformService_UpdateKeepsCoreBuiltinProtocolLockedAfterRename(t *testing.T) {
	repo := &platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			"grok-custom": {
				Slug:         "grok-custom",
				DisplayName:  "Grok Custom",
				Protocol:     PlatformProtocolOpenAICompatible,
				BaseURL:      "https://official.grok.test",
				AuthModes:    []string{AccountTypeOAuth, AccountTypeAPIKey},
				Capabilities: []string{PlatformCapabilityResponses, PlatformCapabilityChatCompletions},
				Enabled:      true,
				Builtin:      true,
			},
		},
	}
	svc := NewPlatformService(repo)

	updated, err := svc.Update(context.Background(), "grok-custom", Platform{
		Slug:        "grok-custom",
		DisplayName: "Grok Custom",
		Protocol:    PlatformProtocolGemini,
		BaseURL:     "https://custom.grok.test",
		AuthModes:   []string{AccountTypeAPIKey},
		Enabled:     true,
	})

	require.NoError(t, err)
	require.Empty(t, repo.renamed)
	require.Equal(t, PlatformProtocolOpenAICompatible, updated.Protocol)
	require.Equal(t, "https://official.grok.test", updated.BaseURL)
	require.ElementsMatch(t, []string{AccountTypeOAuth, AccountTypeAPIKey}, updated.AuthModes)
}

func TestPlatformService_ListHidesNonBuiltinPlatforms(t *testing.T) {
	repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{
		PlatformOpenAIAudio: {
			Slug: PlatformOpenAIAudio, Builtin: true, Enabled: true,
		},
		"legacy-custom": {
			Slug: "legacy-custom", Protocol: PlatformProtocolOpenAICompatible, Enabled: true,
		},
	}}

	platforms, err := NewPlatformService(repo).List(context.Background(), true)

	require.NoError(t, err)
	require.Len(t, platforms, 1)
	require.Equal(t, PlatformOpenAIAudio, platforms[0].Slug)
}

func TestPlatformService_GetBySlugRejectsNonBuiltinPlatform(t *testing.T) {
	repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{
		"legacy-custom": {
			Slug: "legacy-custom", Protocol: PlatformProtocolOpenAICompatible, Enabled: true,
		},
	}}

	platform, err := NewPlatformService(repo).GetBySlug(context.Background(), "legacy-custom")

	require.ErrorIs(t, err, ErrPlatformNotFound)
	require.Nil(t, platform)
}

func TestPlatformService_UpdateRejectsNonBuiltinPlatform(t *testing.T) {
	repo := &platformRepoStubForPlatformService{platforms: map[string]Platform{
		"legacy-custom": {
			Slug: "legacy-custom", DisplayName: "Legacy", Protocol: PlatformProtocolOpenAICompatible, Enabled: true,
		},
	}}

	updated, err := NewPlatformService(repo).Update(context.Background(), "legacy-custom", Platform{
		Slug: "legacy-custom", DisplayName: "Changed", Protocol: PlatformProtocolOpenAICompatible, Enabled: true,
	})

	require.ErrorIs(t, err, ErrPlatformNotFound)
	require.Nil(t, updated)
	require.Equal(t, "Legacy", repo.platforms["legacy-custom"].DisplayName)
}
