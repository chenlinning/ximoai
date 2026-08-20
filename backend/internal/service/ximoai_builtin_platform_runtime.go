package service

import "strings"

const (
	PlatformKindGrokVideo           = "grok_video"
	PlatformKindOpenAIAudio         = "openai_audio"
	PlatformKindKlingAudio          = "kling_audio"
	PlatformKindVolcengineAgentPlan = "volcengine_agent_plan"
)

const PlatformDefaultBaseURLVolcengineAgentPlan = "https://ark.cn-beijing.volces.com/api/plan/v3"

func NormalizePlatformSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func XimoAIPlatformKindFromSlug(slug string) string {
	switch NormalizePlatformSlug(slug) {
	case PlatformGrokVideo:
		return PlatformKindGrokVideo
	case PlatformOpenAIAudio:
		return PlatformKindOpenAIAudio
	case PlatformKlingAudio:
		return PlatformKindKlingAudio
	case PlatformVolcengineAgentPlan:
		return PlatformKindVolcengineAgentPlan
	default:
		return ""
	}
}

func IsXimoAIMediaPlatformKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case PlatformKindGrokVideo, PlatformKindOpenAIAudio, PlatformKindKlingAudio, PlatformKindVolcengineAgentPlan:
		return true
	default:
		return false
	}
}

func XimoAIPlatformRequiresAPIKeyBaseURL(platform string) bool {
	switch XimoAIPlatformKindFromSlug(platform) {
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
	return XimoAIPlatformKindFromSlug(a.Platform)
}

func (a *Account) IsOpenAIAudio() bool {
	return a != nil && a.PlatformRuntimeKind() == PlatformKindOpenAIAudio
}

func (a *Account) IsKlingAudio() bool {
	return a != nil && a.PlatformRuntimeKind() == PlatformKindKlingAudio
}

func (a *Account) UsesOpenAIAPIKeyProtocol() bool {
	return a != nil && a.IsOpenAIApiKey()
}

func (a *Account) UsesAnthropicAPIKeyProtocol() bool {
	return a != nil && a.Type == AccountTypeAPIKey && NormalizePlatformSlug(a.Platform) == PlatformAnthropic
}
