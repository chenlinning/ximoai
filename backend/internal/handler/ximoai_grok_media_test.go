package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGrokMediaPlatformForAPIKeyKeepsBuiltinAndCustomPoolsSeparate(t *testing.T) {
	require.Equal(t, service.PlatformGrok, grokMediaPlatformForAPIKey(context.Background(), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformGrok},
	}))
	require.Equal(t, service.PlatformGrokVideo, grokMediaPlatformForAPIKey(context.Background(), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformGrokVideo},
	}))
	require.Equal(t, service.PlatformOpenAI, grokMediaPlatformForAPIKey(context.Background(), &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	}))
	require.Equal(t, "renamed-video", grokMediaPlatformForAPIKey(
		service.WithResolvedTargetPlatform(context.Background(), "renamed-video"),
		&service.APIKey{Group: &service.Group{Platform: service.PlatformComposite}},
	))
}
