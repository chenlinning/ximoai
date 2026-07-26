package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	SettingKeyXimoAIModelMetadataOverrides = "ximoai_model_metadata_overrides"
	SettingKeyXimoAIModelBrandOverrides    = "ximoai_model_brand_overrides"
	ximoAIModelMetadataStoreVersion        = 1
	ximoAIModelMetadataMaxStoreBytes       = 256 * 1024
	ximoAIModelBrandMaxLength              = 64
)

const (
	ModelTypeConversation = "conversation"
	ModelTypeEmbedding    = "embedding"
	ModelTypeImage        = "image"
	ModelTypeVideo        = "video"
	ModelTypeTTS          = "tts"
	ModelTypeASR          = "asr"

	ModelInvocationSync          = "sync"
	ModelInvocationStream        = "stream"
	ModelInvocationAsync         = "async"
	ModelInvocationBidirectional = "bidirectional"
	ModelInvocationBatch         = "batch"
)

var ximoAIModelReasoningLevelOptions = []string{
	"none",
	"minimal",
	"low",
	"medium",
	"high",
	"xhigh",
	"max",
}

var (
	ximoAIModelTypeOptions = []string{
		ModelTypeConversation,
		ModelTypeEmbedding,
		ModelTypeImage,
		ModelTypeVideo,
		ModelTypeTTS,
		ModelTypeASR,
	}
	ximoAIModelInvocationModeOptions = []string{
		ModelInvocationSync,
		ModelInvocationStream,
		ModelInvocationAsync,
		ModelInvocationBidirectional,
		ModelInvocationBatch,
	}
)

type ModelMetadataOverride struct {
	Platform          string    `json:"platform"`
	Model             string    `json:"model"`
	Brand             *string   `json:"brand,omitempty"`
	Types             *[]string `json:"types,omitempty"`
	InvocationModes   *[]string `json:"invocation_modes,omitempty"`
	ReasoningLevels   *[]string `json:"reasoning_levels,omitempty"`
	ThinkingSupported *bool     `json:"thinking_supported,omitempty"`
}

type ximoAIModelMetadataStore struct {
	Version   int                              `json:"version"`
	Overrides map[string]ModelMetadataOverride `json:"overrides"`
}

var ximoAIModelMetadataSettingsMu sync.Mutex

func XimoAIModelTypeOptions() []string {
	return append([]string(nil), ximoAIModelTypeOptions...)
}

func XimoAIModelInvocationModeOptions() []string {
	return append([]string(nil), ximoAIModelInvocationModeOptions...)
}

func XimoAIModelReasoningLevelOptions() []string {
	return append([]string(nil), ximoAIModelReasoningLevelOptions...)
}

func ValidateXimoAIModelBrand(brand string) error {
	brand = strings.TrimSpace(brand)
	if brand == "" {
		return errors.New("brand is required")
	}
	if utf8.RuneCountInString(brand) > ximoAIModelBrandMaxLength {
		return fmt.Errorf("brand must not exceed %d characters", ximoAIModelBrandMaxLength)
	}
	for _, value := range brand {
		if unicode.IsControl(value) {
			return errors.New("brand must not contain control characters")
		}
	}
	return nil
}

func (s *SettingService) GetXimoAIModelMetadataOverrides(
	ctx context.Context,
	targets []ModelMetadataOverride,
) ([]*ModelMetadataOverride, error) {
	store, err := s.loadXimoAIModelMetadataStore(ctx)
	if err != nil {
		return nil, err
	}
	overrides := make([]*ModelMetadataOverride, len(targets))
	for index, target := range targets {
		override, ok := store.Overrides[ximoAIModelMetadataOverrideKey(target.Platform, target.Model)]
		if !ok {
			continue
		}
		copy := cloneXimoAIModelMetadataOverride(override)
		overrides[index] = &copy
	}
	return overrides, nil
}

func (s *SettingService) SaveXimoAIModelMetadataOverride(ctx context.Context, override ModelMetadataOverride) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository is not configured")
	}
	normalized, err := normalizeXimoAIModelMetadataOverride(override)
	if err != nil {
		return err
	}
	if normalized.Platform == "" || normalized.Model == "" {
		return errors.New("platform and model are required")
	}
	if normalized.Brand == nil && normalized.Types == nil && normalized.InvocationModes == nil && normalized.ReasoningLevels == nil && normalized.ThinkingSupported == nil {
		return errors.New("at least one metadata override is required")
	}

	ximoAIModelMetadataSettingsMu.Lock()
	defer ximoAIModelMetadataSettingsMu.Unlock()
	store, err := s.loadXimoAIModelMetadataStore(ctx)
	if err != nil {
		return err
	}
	store.Overrides[ximoAIModelMetadataOverrideKey(normalized.Platform, normalized.Model)] = normalized
	if err := s.saveXimoAIModelMetadataStore(ctx, store); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) DeleteXimoAIModelMetadataOverride(ctx context.Context, platform, model string) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository is not configured")
	}

	ximoAIModelMetadataSettingsMu.Lock()
	defer ximoAIModelMetadataSettingsMu.Unlock()
	store, err := s.loadXimoAIModelMetadataStore(ctx)
	if err != nil {
		return err
	}
	delete(store.Overrides, ximoAIModelMetadataOverrideKey(platform, model))
	if err := s.saveXimoAIModelMetadataStore(ctx, store); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) loadXimoAIModelMetadataStore(ctx context.Context) (ximoAIModelMetadataStore, error) {
	empty := ximoAIModelMetadataStore{
		Version:   ximoAIModelMetadataStoreVersion,
		Overrides: map[string]ModelMetadataOverride{},
	}
	if s == nil || s.settingRepo == nil {
		return empty, errors.New("setting repository is not configured")
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyXimoAIModelMetadataOverrides)
	if err != nil {
		if !errors.Is(err, ErrSettingNotFound) && err != ErrSettingNotFound {
			return empty, fmt.Errorf("load model metadata overrides: %w", err)
		}
		raw, err = s.settingRepo.GetValue(ctx, SettingKeyXimoAIModelBrandOverrides)
		if err != nil {
			if errors.Is(err, ErrSettingNotFound) || err == ErrSettingNotFound {
				return empty, nil
			}
			return empty, fmt.Errorf("load legacy model brand overrides: %w", err)
		}
	}
	if len(raw) > ximoAIModelMetadataMaxStoreBytes {
		return empty, errors.New("model metadata overrides exceed the storage limit")
	}
	var store ximoAIModelMetadataStore
	if err := json.Unmarshal([]byte(raw), &store); err != nil {
		return empty, fmt.Errorf("decode model metadata overrides: %w", err)
	}
	if store.Version != ximoAIModelMetadataStoreVersion {
		return empty, fmt.Errorf("unsupported model metadata store version %d", store.Version)
	}
	if store.Overrides == nil {
		store.Overrides = map[string]ModelMetadataOverride{}
	}
	return store, nil
}

func (s *SettingService) saveXimoAIModelMetadataStore(ctx context.Context, store ximoAIModelMetadataStore) error {
	store.Version = ximoAIModelMetadataStoreVersion
	if store.Overrides == nil {
		store.Overrides = map[string]ModelMetadataOverride{}
	}
	raw, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("encode model metadata overrides: %w", err)
	}
	if len(raw) > ximoAIModelMetadataMaxStoreBytes {
		return errors.New("model metadata overrides exceed the storage limit")
	}
	if err := s.settingRepo.Set(ctx, SettingKeyXimoAIModelMetadataOverrides, string(raw)); err != nil {
		return fmt.Errorf("save model metadata overrides: %w", err)
	}
	return nil
}

func ximoAIModelMetadataOverrideKey(platform, model string) string {
	value := strings.ToLower(strings.TrimSpace(platform)) + "\x00" + strings.TrimSpace(model)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func normalizeXimoAIModelMetadataOverride(override ModelMetadataOverride) (ModelMetadataOverride, error) {
	override.Platform = strings.ToLower(strings.TrimSpace(override.Platform))
	override.Model = strings.TrimSpace(override.Model)
	if override.Brand != nil {
		brand := strings.TrimSpace(*override.Brand)
		if err := ValidateXimoAIModelBrand(brand); err != nil {
			return ModelMetadataOverride{}, err
		}
		override.Brand = &brand
	}
	var err error
	override.Types, err = normalizeXimoAIModelMetadataValues(override.Types, ximoAIModelTypeOptions, "model type")
	if err != nil {
		return ModelMetadataOverride{}, err
	}
	override.InvocationModes, err = normalizeXimoAIModelMetadataValues(override.InvocationModes, ximoAIModelInvocationModeOptions, "invocation mode")
	if err != nil {
		return ModelMetadataOverride{}, err
	}
	override.ReasoningLevels, err = normalizeXimoAIModelMetadataValues(override.ReasoningLevels, ximoAIModelReasoningLevelOptions, "reasoning level")
	if err != nil {
		return ModelMetadataOverride{}, err
	}
	return override, nil
}

func normalizeXimoAIModelMetadataValues(values *[]string, allowed []string, label string) (*[]string, error) {
	if values == nil {
		return nil, nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}
	normalized := make([]string, 0, len(*values))
	seen := make(map[string]struct{}, len(*values))
	for _, value := range *values {
		value = strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowedSet[value]; !ok {
			return nil, fmt.Errorf("unsupported %s %q", label, value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("at least one %s is required", label)
	}
	return &normalized, nil
}

func cloneXimoAIModelMetadataOverride(override ModelMetadataOverride) ModelMetadataOverride {
	copy := override
	if override.Brand != nil {
		brand := *override.Brand
		copy.Brand = &brand
	}
	if override.Types != nil {
		types := append([]string(nil), (*override.Types)...)
		copy.Types = &types
	}
	if override.InvocationModes != nil {
		modes := append([]string(nil), (*override.InvocationModes)...)
		copy.InvocationModes = &modes
	}
	if override.ReasoningLevels != nil {
		levels := append([]string(nil), (*override.ReasoningLevels)...)
		copy.ReasoningLevels = &levels
	}
	if override.ThinkingSupported != nil {
		thinkingSupported := *override.ThinkingSupported
		copy.ThinkingSupported = &thinkingSupported
	}
	return copy
}
