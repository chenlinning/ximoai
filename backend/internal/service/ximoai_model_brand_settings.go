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
	SettingKeyXimoAIModelBrandOverrides = "ximoai_model_brand_overrides"
	ximoAIModelBrandStoreVersion        = 1
	ximoAIModelBrandMaxStoreBytes       = 256 * 1024
	ximoAIModelBrandMaxLength           = 64
)

type ModelBrandOverride struct {
	Platform string `json:"platform"`
	Model    string `json:"model"`
	Brand    string `json:"brand"`
}

type ximoAIModelBrandStore struct {
	Version   int                           `json:"version"`
	Overrides map[string]ModelBrandOverride `json:"overrides"`
}

var ximoAIModelBrandSettingsMu sync.Mutex

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

func (s *SettingService) GetXimoAIModelBrandOverrides(
	ctx context.Context,
	targets []ModelBrandOverride,
) ([]*ModelBrandOverride, error) {
	store, err := s.loadXimoAIModelBrandStore(ctx)
	if err != nil {
		return nil, err
	}
	overrides := make([]*ModelBrandOverride, len(targets))
	for i := range targets {
		target := targets[i]
		override, ok := store.Overrides[ximoAIModelBrandOverrideKey(target.Platform, target.Model)]
		if !ok {
			continue
		}
		overrideCopy := override
		overrides[i] = &overrideCopy
	}
	return overrides, nil
}

func (s *SettingService) SaveXimoAIModelBrandOverride(ctx context.Context, override ModelBrandOverride) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository is not configured")
	}
	override = normalizeXimoAIModelBrandOverride(override)
	if override.Platform == "" || override.Model == "" {
		return errors.New("platform and model are required")
	}
	if err := ValidateXimoAIModelBrand(override.Brand); err != nil {
		return err
	}

	ximoAIModelBrandSettingsMu.Lock()
	defer ximoAIModelBrandSettingsMu.Unlock()
	store, err := s.loadXimoAIModelBrandStore(ctx)
	if err != nil {
		return err
	}
	store.Overrides[ximoAIModelBrandOverrideKey(override.Platform, override.Model)] = override
	if err := s.saveXimoAIModelBrandStore(ctx, store); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) DeleteXimoAIModelBrandOverride(ctx context.Context, platform, model string) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository is not configured")
	}

	ximoAIModelBrandSettingsMu.Lock()
	defer ximoAIModelBrandSettingsMu.Unlock()
	store, err := s.loadXimoAIModelBrandStore(ctx)
	if err != nil {
		return err
	}
	delete(store.Overrides, ximoAIModelBrandOverrideKey(platform, model))
	if err := s.saveXimoAIModelBrandStore(ctx, store); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) loadXimoAIModelBrandStore(ctx context.Context) (ximoAIModelBrandStore, error) {
	empty := ximoAIModelBrandStore{
		Version:   ximoAIModelBrandStoreVersion,
		Overrides: map[string]ModelBrandOverride{},
	}
	if s == nil || s.settingRepo == nil {
		return empty, errors.New("setting repository is not configured")
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyXimoAIModelBrandOverrides)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) || err == ErrSettingNotFound {
			return empty, nil
		}
		return empty, fmt.Errorf("load model brand overrides: %w", err)
	}
	if len(raw) > ximoAIModelBrandMaxStoreBytes {
		return empty, errors.New("model brand overrides exceed the storage limit")
	}
	var store ximoAIModelBrandStore
	if err := json.Unmarshal([]byte(raw), &store); err != nil {
		return empty, fmt.Errorf("decode model brand overrides: %w", err)
	}
	if store.Version != ximoAIModelBrandStoreVersion {
		return empty, fmt.Errorf("unsupported model brand store version %d", store.Version)
	}
	if store.Overrides == nil {
		store.Overrides = map[string]ModelBrandOverride{}
	}
	return store, nil
}

func (s *SettingService) saveXimoAIModelBrandStore(ctx context.Context, store ximoAIModelBrandStore) error {
	store.Version = ximoAIModelBrandStoreVersion
	if store.Overrides == nil {
		store.Overrides = map[string]ModelBrandOverride{}
	}
	raw, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("encode model brand overrides: %w", err)
	}
	if len(raw) > ximoAIModelBrandMaxStoreBytes {
		return errors.New("model brand overrides exceed the storage limit")
	}
	if err := s.settingRepo.Set(ctx, SettingKeyXimoAIModelBrandOverrides, string(raw)); err != nil {
		return fmt.Errorf("save model brand overrides: %w", err)
	}
	return nil
}

func ximoAIModelBrandOverrideKey(platform, model string) string {
	value := strings.ToLower(strings.TrimSpace(platform)) + "\x00" + strings.TrimSpace(model)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func normalizeXimoAIModelBrandOverride(override ModelBrandOverride) ModelBrandOverride {
	override.Platform = strings.ToLower(strings.TrimSpace(override.Platform))
	override.Model = strings.TrimSpace(override.Model)
	override.Brand = strings.TrimSpace(override.Brand)
	return override
}
