package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ximoAICustomOpenAIAPIKeyAccount() *Account {
	return &Account{
		Platform: "acme-openai",
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":           "sk-test",
			"base_url":          "https://api.acme.example/v1",
			"platform_protocol": PlatformProtocolOpenAICompatible,
		},
	}
}

func TestXimoAICustomOpenAIAPIKeyProfileBoundary(t *testing.T) {
	custom := ximoAICustomOpenAIAPIKeyAccount()
	require.True(t, custom.UsesOpenAIAPIKeyProtocol())
	require.True(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}).UsesOpenAIAPIKeyProtocol())
	require.False(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}).UsesOpenAIAPIKeyProtocol())
	require.False(t, (&Account{Platform: "custom-gemini", Type: AccountTypeAPIKey, Credentials: map[string]any{
		"platform_protocol": PlatformProtocolGemini,
	}}).UsesOpenAIAPIKeyProtocol())
	require.False(t, (&Account{Platform: "unknown-custom", Type: AccountTypeAPIKey}).UsesOpenAIAPIKeyProtocol())

	for _, platform := range []string{PlatformGrokVideo, PlatformOpenAIAudio, PlatformKlingAudio} {
		require.False(t, (&Account{Platform: platform, Type: AccountTypeAPIKey, Credentials: map[string]any{
			"platform_protocol": PlatformProtocolOpenAICompatible,
		}}).UsesOpenAIAPIKeyProtocol(), platform)
	}
}

func TestXimoAICustomOpenAIAPIKeyProfileCapabilities(t *testing.T) {
	account := ximoAICustomOpenAIAPIKeyAccount()
	account.Extra = map[string]any{
		"openai_passthrough":                             true,
		"openai_long_context_billing_enabled":            true,
		"openai_compact_mode":                            OpenAICompactModeForceOn,
		"openai_compact_supported":                       false,
		featureKeyCodexImageGenerationBridge:             true,
		featureKeyCodexImageGenerationExplicitToolPolicy: codexImageGenerationExplicitToolPolicyStrip,
	}
	account.Credentials["openai_capabilities"] = []any{"embeddings"}
	account.Credentials[credKeyHeaderOverrideEnabled] = true
	account.Credentials[credKeyHeaderOverrides] = map[string]any{"x-tenant": "acme"}

	require.True(t, account.IsOpenAIPassthroughEnabled())
	require.True(t, account.IsOpenAILongContextBillingEnabled())
	require.Equal(t, OpenAICompactModeForceOn, account.GetOpenAICompactMode())
	require.True(t, account.AllowsOpenAICompact())
	require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityEmbeddings))
	require.True(t, account.IsHeaderOverrideEligible())
	require.Equal(t, map[string]string{"x-tenant": "acme"}, account.GetHeaderOverrides())
	require.True(t, isUpstreamBillingProbeAccount(account))
	require.True(t, openAICompactRequiredForAccount(account, true))
	require.Equal(t, 2, openAICompactSupportTier(account))
	require.Equal(t, true, *account.CodexImageGenerationBridgeOverride())
	require.Equal(t, codexImageGenerationExplicitToolPolicyStrip, account.CodexImageGenerationExplicitToolPolicy())

	account.Extra["openai_compact_mode"] = OpenAICompactModeForceOff
	require.Equal(t, 0, openAICompactSupportTier(account))
}
