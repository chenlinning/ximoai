package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestXimoAIOpenAICompatibleRequestPlatformPreservesCustomGroup(t *testing.T) {
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
		{name: "custom openai compatible", apiKey: &service.APIKey{Group: &service.Group{Platform: "volcengine"}}, expected: "volcengine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, openAICompatibleRequestPlatform(tt.apiKey))
		})
	}
}
