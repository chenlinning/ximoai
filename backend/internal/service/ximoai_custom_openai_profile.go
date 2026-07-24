package service

import "strings"

// UsesOpenAIAPIKeyProtocol reports whether an account uses the standard OpenAI
// API key capability profile. Identity-based OpenAI accounts are intentionally
// excluded because custom platforms must never inherit OAuth-only behavior.
func (a *Account) UsesOpenAIAPIKeyProtocol() bool {
	if a == nil || a.Type != AccountTypeAPIKey {
		return false
	}
	if a.IsOpenAIApiKey() {
		return true
	}
	if a.IsGrokVideo() || a.IsOpenAIAudio() || a.IsKlingAudio() {
		return false
	}
	switch NormalizePlatformSlug(a.Platform) {
	case "", PlatformAnthropic, PlatformGemini, PlatformAntigravity, PlatformGrok,
		PlatformGrokVideo, PlatformOpenAIAudio, PlatformKlingAudio:
		return false
	}
	protocol := strings.ToLower(strings.TrimSpace(a.GetCredential("platform_protocol")))
	return protocol == PlatformProtocolOpenAI || protocol == PlatformProtocolOpenAICompatible
}
