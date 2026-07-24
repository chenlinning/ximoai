package service

import "strings"

const (
	PlatformKindCredentialKey = "platform_kind"
	PlatformKindGrokVideo     = "grok_video"
	PlatformKindOpenAIAudio   = "openai_audio"
	PlatformKindKlingAudio    = "kling_audio"
)

func XimoAIPlatformKindFromLegacySlug(slug string) string {
	switch NormalizePlatformSlug(slug) {
	case PlatformGrokVideo:
		return PlatformKindGrokVideo
	case PlatformOpenAIAudio:
		return PlatformKindOpenAIAudio
	case PlatformKlingAudio:
		return PlatformKindKlingAudio
	default:
		return ""
	}
}

func IsXimoAIMediaPlatformKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case PlatformKindGrokVideo, PlatformKindOpenAIAudio, PlatformKindKlingAudio:
		return true
	default:
		return false
	}
}

func (p Platform) RuntimeKind() string {
	if kind := strings.ToLower(strings.TrimSpace(p.Kind)); kind != "" {
		return kind
	}
	return XimoAIPlatformKindFromLegacySlug(p.Slug)
}

func (p Platform) RequiresAPIKeyBaseURL() bool {
	switch p.RuntimeKind() {
	case PlatformKindGrokVideo, PlatformKindKlingAudio:
		return true
	default:
		return false
	}
}

func (a *Account) PlatformRuntimeKind() string {
	if a == nil {
		return ""
	}
	if kind := strings.ToLower(strings.TrimSpace(a.GetCredential(PlatformKindCredentialKey))); kind != "" {
		return kind
	}
	return XimoAIPlatformKindFromLegacySlug(a.Platform)
}

func (a *Account) IsOpenAIAudio() bool {
	return a != nil && a.PlatformRuntimeKind() == PlatformKindOpenAIAudio
}

func (a *Account) IsKlingAudio() bool {
	return a != nil && a.PlatformRuntimeKind() == PlatformKindKlingAudio
}
