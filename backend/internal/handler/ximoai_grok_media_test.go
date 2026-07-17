package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGrokMediaPlatformForAPIKeyKeepsBuiltinAndCustomPoolsSeparate(t *testing.T) {
	require.Equal(t, service.PlatformGrok, grokMediaPlatformForAPIKey(&service.APIKey{
		Group: &service.Group{Platform: service.PlatformGrok},
	}))
	require.Equal(t, service.PlatformGrokVideo, grokMediaPlatformForAPIKey(&service.APIKey{
		Group: &service.Group{Platform: service.PlatformGrokVideo},
	}))
	require.Equal(t, service.PlatformGrok, grokMediaPlatformForAPIKey(&service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	}))
}
