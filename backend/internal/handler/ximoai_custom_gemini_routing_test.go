package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestXimoAICustomGeminiAPIKeyUsesGeminiCompatibilityForwarder(t *testing.T) {
	h := &GatewayHandler{}

	require.True(t, h.isGeminiProtocolAccount(&service.Account{
		Platform: "acme-gemini",
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"platform_protocol": service.PlatformProtocolGemini,
		},
	}))
	require.True(t, h.isGeminiProtocolAccount(&service.Account{
		Platform: service.PlatformGemini,
		Type:     service.AccountTypeOAuth,
	}))
	require.False(t, h.isGeminiProtocolAccount(&service.Account{
		Platform: "acme-anthropic",
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"platform_protocol": service.PlatformProtocolAnthropic,
		},
	}))
}
