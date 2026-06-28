package service

import "time"

const (
	PlatformProtocolNative           = "native"
	PlatformProtocolOpenAI           = "openai"
	PlatformProtocolOpenAICompatible = "openai_compatible"
	PlatformProtocolAnthropic        = "anthropic"
	PlatformProtocolGemini           = "gemini"

	PlatformCapabilityResponses       = "responses"
	PlatformCapabilityChatCompletions = "chat_completions"
	PlatformCapabilityImages          = "images"
	PlatformCapabilityVideos          = "videos"
	PlatformCapabilityAudio           = "audio"
	PlatformCapabilityRealtime        = "realtime"
	PlatformCapabilityCodex           = "codex"
	PlatformCapabilityMessages        = "messages"
	PlatformCapabilityNativeGemini    = "native_gemini"
)

const (
	PlatformDefaultBaseURLAnthropic   = "https://api.anthropic.com"
	PlatformDefaultBaseURLOpenAI      = "https://api.openai.com"
	PlatformDefaultBaseURLGemini      = "https://generativelanguage.googleapis.com"
	PlatformDefaultBaseURLAntigravity = "https://cloudcode-pa.googleapis.com"
	PlatformDefaultBaseURLGrok        = "https://api.x.ai/v1"
)

type Platform struct {
	Slug         string
	DisplayName  string
	Protocol     string
	BaseURL      string
	AuthModes    []string
	Capabilities []string
	Color        string
	Enabled      bool
	Builtin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (p Platform) SupportsAuthMode(mode string) bool {
	for _, item := range p.AuthModes {
		if item == mode {
			return true
		}
	}
	return false
}

func (p Platform) SupportsCapability(capability string) bool {
	for _, item := range p.Capabilities {
		if item == capability {
			return true
		}
	}
	return false
}

func (p Platform) IsOpenAICompatible() bool {
	return p.Protocol == PlatformProtocolOpenAI || p.Protocol == PlatformProtocolOpenAICompatible
}

func (p Platform) IsAnthropicCompatible() bool {
	return p.Protocol == PlatformProtocolAnthropic ||
		(p.Protocol == PlatformProtocolNative && p.Slug == PlatformAnthropic)
}

func (p Platform) IsGeminiCompatible() bool {
	return p.Protocol == PlatformProtocolGemini ||
		(p.Protocol == PlatformProtocolNative && p.Slug == PlatformGemini)
}

func (p Platform) IsOfficialOpenAI() bool {
	return p.Slug == PlatformOpenAI && p.Protocol == PlatformProtocolOpenAI
}
