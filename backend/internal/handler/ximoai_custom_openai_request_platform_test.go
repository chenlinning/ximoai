package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestXimoAIOpenAICompatibleRequestPlatformPreservesFixedBuiltinGroup(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   *service.APIKey
		expected string
	}{
		{name: "missing api key", expected: service.PlatformOpenAI},
		{name: "missing group", apiKey: &service.APIKey{}, expected: service.PlatformOpenAI},
		{name: "empty platform", apiKey: &service.APIKey{Group: &service.Group{}}, expected: service.PlatformOpenAI},
		{name: "official openai", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformOpenAI}}, expected: service.PlatformOpenAI},
		{name: "official grok", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformGrok}}, expected: service.PlatformGrok},
		{name: "official anthropic keeps fallback", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformAnthropic}}, expected: service.PlatformOpenAI},
		{name: "official gemini keeps fallback", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformGemini}}, expected: service.PlatformOpenAI},
		{name: "official antigravity keeps fallback", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformAntigravity}}, expected: service.PlatformOpenAI},
		{name: "grok video", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformGrokVideo}}, expected: service.PlatformGrokVideo},
		{name: "openai audio", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformOpenAIAudio}}, expected: service.PlatformOpenAIAudio},
		{name: "kling audio", apiKey: &service.APIKey{Group: &service.Group{Platform: service.PlatformKlingAudio}}, expected: service.PlatformKlingAudio},
		{name: "unknown platform falls back", apiKey: &service.APIKey{Group: &service.Group{Platform: "volcengine"}}, expected: service.PlatformOpenAI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, openAICompatibleRequestPlatform(context.Background(), tt.apiKey))
		})
	}
}
