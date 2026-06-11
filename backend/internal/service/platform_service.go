package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var platformSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}[a-z0-9]$`)
var customPlatformDefaultColors = []string{
	"#0F766E",
	"#2563EB",
	"#7C3AED",
	"#DB2777",
	"#EA580C",
	"#059669",
	"#4F46E5",
	"#0891B2",
}

var (
	ErrPlatformNotFound = infraerrors.NotFound("PLATFORM_NOT_FOUND", "platform not found")
	ErrPlatformInvalid  = infraerrors.BadRequest("INVALID_PLATFORM", "invalid platform")
)

type PlatformRepository interface {
	List(ctx context.Context, includeDisabled bool) ([]Platform, error)
	GetBySlug(ctx context.Context, slug string) (*Platform, error)
	Create(ctx context.Context, platform *Platform) error
	Update(ctx context.Context, platform *Platform) error
	Delete(ctx context.Context, slug string) error
	Usage(ctx context.Context, slug string) (PlatformUsage, error)
}

type PlatformService struct {
	repo PlatformRepository
}

type PlatformUsage struct {
	AccountCount             int64
	GroupCount               int64
	ChannelPricingCount      int64
	AccountStatsPricingCount int64
	ChannelMappingCount      int64
}

func (u PlatformUsage) Total() int64 {
	return u.AccountCount + u.GroupCount + u.ChannelPricingCount + u.AccountStatsPricingCount + u.ChannelMappingCount
}

func NewPlatformService(repo PlatformRepository) *PlatformService {
	return &PlatformService{repo: repo}
}

func (s *PlatformService) List(ctx context.Context, includeDisabled bool) ([]Platform, error) {
	if s == nil || s.repo == nil {
		return builtinPlatforms(includeDisabled), nil
	}
	platforms, err := s.repo.List(ctx, includeDisabled)
	if err != nil {
		return nil, err
	}
	sortPlatforms(platforms)
	return platforms, nil
}

func (s *PlatformService) GetBySlug(ctx context.Context, slug string) (*Platform, error) {
	slug = NormalizePlatformSlug(slug)
	if slug == "" {
		return nil, ErrPlatformInvalid
	}
	if s == nil || s.repo == nil {
		for _, p := range builtinPlatforms(true) {
			if p.Slug == slug {
				cp := p
				return &cp, nil
			}
		}
		return nil, ErrPlatformNotFound
	}
	return s.repo.GetBySlug(ctx, slug)
}

func (s *PlatformService) Create(ctx context.Context, input Platform) (*Platform, error) {
	platform, err := normalizePlatformInput(input, false)
	if err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("platform repository not configured")
	}
	if err := s.repo.Create(ctx, platform); err != nil {
		return nil, err
	}
	return s.repo.GetBySlug(ctx, platform.Slug)
}

func (s *PlatformService) Update(ctx context.Context, slug string, input Platform) (*Platform, error) {
	slug = NormalizePlatformSlug(slug)
	if slug == "" {
		return nil, ErrPlatformInvalid
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("platform repository not configured")
	}
	current, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if current.Builtin {
		input.Protocol = current.Protocol
		input.AuthModes = current.AuthModes
		input.Capabilities = current.Capabilities
		if !builtinPlatformBaseURLEditable(current.Slug) {
			input.BaseURL = current.BaseURL
		}
	}
	input.Slug = current.Slug
	input.Builtin = current.Builtin
	platform, err := normalizePlatformInput(input, true)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, platform); err != nil {
		return nil, err
	}
	return s.repo.GetBySlug(ctx, slug)
}

func (s *PlatformService) Delete(ctx context.Context, slug string) error {
	slug = NormalizePlatformSlug(slug)
	if slug == "" {
		return ErrPlatformInvalid
	}
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if platform.Builtin {
		return infraerrors.BadRequest("BUILTIN_PLATFORM_READONLY", "builtin platforms cannot be deleted")
	}
	usage, err := s.repo.Usage(ctx, slug)
	if err != nil {
		return err
	}
	if usage.Total() > 0 {
		return infraerrors.Conflict(
			"PLATFORM_IN_USE",
			fmt.Sprintf(
				"platform %s is in use by %d accounts, %d groups, %d channel pricing rules, %d account stats pricing rules, and %d channel mappings",
				slug,
				usage.AccountCount,
				usage.GroupCount,
				usage.ChannelPricingCount,
				usage.AccountStatsPricingCount,
				usage.ChannelMappingCount,
			),
		)
	}
	return s.repo.Delete(ctx, slug)
}

func (s *PlatformService) IsOpenAICompatible(ctx context.Context, slug string) bool {
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil || platform == nil || !platform.Enabled {
		return false
	}
	return platform.IsOpenAICompatible()
}

func (s *PlatformService) IsAnthropicCompatible(ctx context.Context, slug string) bool {
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil || platform == nil || !platform.Enabled {
		return false
	}
	return platform.IsAnthropicCompatible()
}

func (s *PlatformService) IsGeminiCompatible(ctx context.Context, slug string) bool {
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil || platform == nil || !platform.Enabled {
		return false
	}
	return platform.IsGeminiCompatible()
}

func (s *PlatformService) SupportsCapability(ctx context.Context, slug string, capability string) bool {
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil || platform == nil || !platform.Enabled {
		return false
	}
	return platform.SupportsCapability(capability)
}

func (s *PlatformService) ValidateAccountPlatform(ctx context.Context, platformSlug, accountType string) error {
	platform, err := s.GetBySlug(ctx, platformSlug)
	if err != nil {
		return err
	}
	if !platform.Enabled {
		return infraerrors.BadRequest("PLATFORM_DISABLED", fmt.Sprintf("platform %s is disabled", platform.Slug))
	}
	if !platform.SupportsAuthMode(accountType) {
		return infraerrors.BadRequest("UNSUPPORTED_PLATFORM_AUTH_MODE",
			fmt.Sprintf("platform %s does not support account type %s", platform.Slug, accountType))
	}
	if !platform.Builtin && accountType != AccountTypeAPIKey {
		return infraerrors.BadRequest("CUSTOM_PLATFORM_APIKEY_ONLY", "custom platforms only support API Key accounts")
	}
	return nil
}

func (s *PlatformService) ValidatePlatformSlug(ctx context.Context, slug string) error {
	platform, err := s.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if !platform.Enabled {
		return infraerrors.BadRequest("PLATFORM_DISABLED", fmt.Sprintf("platform %s is disabled", platform.Slug))
	}
	return nil
}

func NormalizePlatformSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func normalizePlatformInput(input Platform, updating bool) (*Platform, error) {
	input.Slug = NormalizePlatformSlug(input.Slug)
	input.Protocol = strings.ToLower(strings.TrimSpace(input.Protocol))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.Color = strings.TrimSpace(input.Color)
	input.AuthModes = normalizeStringSet(input.AuthModes)
	input.Capabilities = normalizeStringSet(input.Capabilities)

	if !platformSlugPattern.MatchString(input.Slug) {
		return nil, infraerrors.BadRequest("INVALID_PLATFORM_SLUG", "platform slug must be 3-64 chars using lowercase letters, digits, hyphen or underscore")
	}
	if input.DisplayName == "" {
		return nil, infraerrors.BadRequest("INVALID_PLATFORM_NAME", "display_name is required")
	}
	if input.Color == "" {
		input.Color = defaultCustomPlatformColor(input.Slug)
	}
	switch input.Protocol {
	case PlatformProtocolNative, PlatformProtocolOpenAI, PlatformProtocolOpenAICompatible, PlatformProtocolAnthropic, PlatformProtocolGemini:
	default:
		return nil, infraerrors.BadRequest("INVALID_PLATFORM_PROTOCOL", "protocol must be native, openai, openai_compatible, anthropic, or gemini")
	}
	if !input.Builtin && !isCustomPlatformProtocol(input.Protocol) {
		return nil, infraerrors.BadRequest("INVALID_CUSTOM_PLATFORM_PROTOCOL", "custom platforms must use openai_compatible, anthropic, or gemini protocol")
	}
	if !input.Builtin && isCustomPlatformProtocol(input.Protocol) {
		if input.BaseURL == "" {
			return nil, infraerrors.BadRequest("INVALID_PLATFORM_BASE_URL", "base_url is required for custom platforms")
		}
		if _, err := url.ParseRequestURI(input.BaseURL); err != nil {
			return nil, infraerrors.BadRequest("INVALID_PLATFORM_BASE_URL", "base_url must be a valid URL")
		}
		input.AuthModes = []string{AccountTypeAPIKey}
		if len(input.Capabilities) == 0 {
			input.Capabilities = defaultCustomPlatformCapabilities(input.Protocol)
		}
	}
	if input.Protocol == PlatformProtocolOpenAI {
		input.Slug = PlatformOpenAI
		input.BaseURL = PlatformDefaultBaseURLOpenAI
	}
	if len(input.AuthModes) == 0 {
		return nil, infraerrors.BadRequest("INVALID_PLATFORM_AUTH_MODES", "auth_modes is required")
	}
	if !updating && input.Enabled == false {
		input.Enabled = true
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now()
	}
	input.UpdatedAt = time.Now()
	return &input, nil
}

func isCustomPlatformProtocol(protocol string) bool {
	switch protocol {
	case PlatformProtocolOpenAICompatible, PlatformProtocolAnthropic, PlatformProtocolGemini:
		return true
	default:
		return false
	}
}

func builtinPlatformBaseURLEditable(slug string) bool {
	switch NormalizePlatformSlug(slug) {
	case PlatformGrok, PlatformKlingAudio:
		return true
	default:
		return false
	}
}

func defaultCustomPlatformCapabilities(protocol string) []string {
	switch protocol {
	case PlatformProtocolAnthropic:
		return []string{PlatformCapabilityMessages}
	case PlatformProtocolGemini:
		return []string{PlatformCapabilityMessages, PlatformCapabilityNativeGemini}
	default:
		return []string{
			PlatformCapabilityResponses,
			PlatformCapabilityChatCompletions,
			PlatformCapabilityImages,
			PlatformCapabilityVideos,
			PlatformCapabilityAudio,
			PlatformCapabilityRealtime,
		}
	}
}

func defaultCustomPlatformColor(slug string) string {
	if len(customPlatformDefaultColors) == 0 {
		return "#64748B"
	}
	hash := 0
	for _, ch := range slug {
		hash = hash*31 + int(ch)
	}
	if hash < 0 {
		hash = -hash
	}
	return customPlatformDefaultColors[hash%len(customPlatformDefaultColors)]
}

func normalizeStringSet(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.ToLower(strings.TrimSpace(value))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func builtinPlatforms(includeDisabled bool) []Platform {
	now := time.Now()
	platforms := []Platform{
		{
			Slug:         PlatformAnthropic,
			DisplayName:  "Anthropic",
			Protocol:     PlatformProtocolNative,
			BaseURL:      PlatformDefaultBaseURLAnthropic,
			AuthModes:    []string{AccountTypeOAuth, AccountTypeSetupToken, AccountTypeAPIKey, AccountTypeBedrock},
			Capabilities: []string{PlatformCapabilityMessages},
			Color:        "#D97706",
			Enabled:      true,
			Builtin:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			Slug:        PlatformOpenAI,
			DisplayName: "OpenAI",
			Protocol:    PlatformProtocolOpenAI,
			BaseURL:     PlatformDefaultBaseURLOpenAI,
			AuthModes:   []string{AccountTypeOAuth, AccountTypeAPIKey},
			Capabilities: []string{
				PlatformCapabilityResponses,
				PlatformCapabilityChatCompletions,
				PlatformCapabilityImages,
				PlatformCapabilityVideos,
				PlatformCapabilityAudio,
				PlatformCapabilityRealtime,
				PlatformCapabilityCodex,
			},
			Color:     "#10A37F",
			Enabled:   true,
			Builtin:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Slug:         PlatformGemini,
			DisplayName:  "Gemini",
			Protocol:     PlatformProtocolNative,
			BaseURL:      PlatformDefaultBaseURLGemini,
			AuthModes:    []string{AccountTypeOAuth, AccountTypeAPIKey, AccountTypeServiceAccount},
			Capabilities: []string{PlatformCapabilityMessages, PlatformCapabilityNativeGemini},
			Color:        "#4285F4",
			Enabled:      true,
			Builtin:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			Slug:         PlatformAntigravity,
			DisplayName:  "Antigravity",
			Protocol:     PlatformProtocolNative,
			BaseURL:      PlatformDefaultBaseURLAntigravity,
			AuthModes:    []string{AccountTypeOAuth, AccountTypeUpstream, AccountTypeAPIKey},
			Capabilities: []string{PlatformCapabilityMessages, PlatformCapabilityNativeGemini},
			Color:        "#7C3AED",
			Enabled:      true,
			Builtin:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			Slug:         PlatformGrok,
			DisplayName:  "Grok",
			Protocol:     PlatformProtocolOpenAICompatible,
			BaseURL:      "",
			AuthModes:    []string{AccountTypeAPIKey},
			Capabilities: []string{PlatformCapabilityVideos},
			Color:        "#111827",
			Enabled:      true,
			Builtin:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			Slug:         PlatformKlingAudio,
			DisplayName:  "可灵 Audio",
			Protocol:     PlatformProtocolOpenAICompatible,
			BaseURL:      "",
			AuthModes:    []string{AccountTypeAPIKey},
			Capabilities: []string{PlatformCapabilityAudio},
			Color:        "#0EA5E9",
			Enabled:      true,
			Builtin:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	if includeDisabled {
		return platforms
	}
	out := make([]Platform, 0, len(platforms))
	for _, p := range platforms {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

func sortPlatforms(platforms []Platform) {
	sort.SliceStable(platforms, func(i, j int) bool {
		if platforms[i].Builtin != platforms[j].Builtin {
			return platforms[i].Builtin
		}
		return strings.ToLower(platforms[i].DisplayName) < strings.ToLower(platforms[j].DisplayName)
	})
}
