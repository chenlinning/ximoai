package service

import "strings"

// UsesAnthropicAPIKeyProtocol reports whether an API-key account should reuse
// the built-in Anthropic API-key behavior without inheriting Anthropic OAuth or
// Bedrock capabilities.
func (a *Account) UsesAnthropicAPIKeyProtocol() bool {
	if a == nil || a.Type != AccountTypeAPIKey {
		return false
	}
	if NormalizePlatformSlug(a.Platform) == PlatformAnthropic {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(a.GetCredential("platform_protocol")), PlatformProtocolAnthropic)
}
