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
)

const (
	SettingKeyXimoAIModelAPIDocsBindings = "ximoai_model_api_docs_bindings"
	ximoAIModelAPIDocsStoreVersion       = 1
	ximoAIModelAPIDocsMaxStoreBytes      = 256 * 1024
)

type ModelAPIDocsEndpointBinding struct {
	Profile  string   `json:"profile"`
	Variants []string `json:"variants"`
}

type ModelAPIDocsCategoryBinding struct {
	Category  string                        `json:"category"`
	Endpoints []ModelAPIDocsEndpointBinding `json:"endpoints"`
}

type ModelAPIDocsBinding struct {
	Platform   string                        `json:"platform"`
	Protocol   string                        `json:"protocol"`
	Model      string                        `json:"model"`
	Categories []ModelAPIDocsCategoryBinding `json:"categories"`
}

type ximoAIModelAPIDocsStore struct {
	Version  int                            `json:"version"`
	Bindings map[string]ModelAPIDocsBinding `json:"bindings"`
}

var ximoAIModelAPIDocsSettingsMu sync.Mutex

func (s *SettingService) GetXimoAIModelAPIDocsBinding(
	ctx context.Context,
	platform string,
	protocol string,
	model string,
) (*ModelAPIDocsBinding, error) {
	bindings, err := s.GetXimoAIModelAPIDocsBindings(ctx, []ModelAPIDocsBinding{{
		Platform: platform,
		Protocol: protocol,
		Model:    model,
	}})
	if err != nil {
		return nil, err
	}
	return bindings[0], nil
}

func (s *SettingService) GetXimoAIModelAPIDocsBindings(
	ctx context.Context,
	targets []ModelAPIDocsBinding,
) ([]*ModelAPIDocsBinding, error) {
	store, err := s.loadXimoAIModelAPIDocsStore(ctx)
	if err != nil {
		return nil, err
	}
	bindings := make([]*ModelAPIDocsBinding, len(targets))
	for i := range targets {
		target := targets[i]
		binding, ok := store.Bindings[ximoAIModelAPIDocsBindingKey(target.Platform, target.Protocol, target.Model)]
		if !ok {
			continue
		}
		bindingCopy := binding
		bindings[i] = &bindingCopy
	}
	return bindings, nil
}

func (s *SettingService) SaveXimoAIModelAPIDocsBinding(ctx context.Context, binding ModelAPIDocsBinding) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository is not configured")
	}
	binding = normalizeModelAPIDocsBinding(binding)
	if binding.Platform == "" || binding.Protocol == "" || binding.Model == "" {
		return errors.New("platform, protocol, and model are required")
	}

	ximoAIModelAPIDocsSettingsMu.Lock()
	defer ximoAIModelAPIDocsSettingsMu.Unlock()
	store, err := s.loadXimoAIModelAPIDocsStore(ctx)
	if err != nil {
		return err
	}
	store.Bindings[ximoAIModelAPIDocsBindingKey(binding.Platform, binding.Protocol, binding.Model)] = binding
	if err := s.saveXimoAIModelAPIDocsStore(ctx, store); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) DeleteXimoAIModelAPIDocsBinding(
	ctx context.Context,
	platform string,
	protocol string,
	model string,
) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository is not configured")
	}

	ximoAIModelAPIDocsSettingsMu.Lock()
	defer ximoAIModelAPIDocsSettingsMu.Unlock()
	store, err := s.loadXimoAIModelAPIDocsStore(ctx)
	if err != nil {
		return err
	}
	delete(store.Bindings, ximoAIModelAPIDocsBindingKey(platform, protocol, model))
	if err := s.saveXimoAIModelAPIDocsStore(ctx, store); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) loadXimoAIModelAPIDocsStore(ctx context.Context) (ximoAIModelAPIDocsStore, error) {
	empty := ximoAIModelAPIDocsStore{
		Version:  ximoAIModelAPIDocsStoreVersion,
		Bindings: map[string]ModelAPIDocsBinding{},
	}
	if s == nil || s.settingRepo == nil {
		return empty, errors.New("setting repository is not configured")
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyXimoAIModelAPIDocsBindings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) || err == ErrSettingNotFound {
			return empty, nil
		}
		return empty, fmt.Errorf("load model API documentation bindings: %w", err)
	}
	if len(raw) > ximoAIModelAPIDocsMaxStoreBytes {
		return empty, errors.New("model API documentation bindings exceed the storage limit")
	}
	var store ximoAIModelAPIDocsStore
	if err := json.Unmarshal([]byte(raw), &store); err != nil {
		return empty, fmt.Errorf("decode model API documentation bindings: %w", err)
	}
	if store.Version != ximoAIModelAPIDocsStoreVersion {
		return empty, fmt.Errorf("unsupported model API documentation bindings version %d", store.Version)
	}
	if store.Bindings == nil {
		store.Bindings = map[string]ModelAPIDocsBinding{}
	}
	return store, nil
}

func (s *SettingService) saveXimoAIModelAPIDocsStore(ctx context.Context, store ximoAIModelAPIDocsStore) error {
	store.Version = ximoAIModelAPIDocsStoreVersion
	if store.Bindings == nil {
		store.Bindings = map[string]ModelAPIDocsBinding{}
	}
	raw, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("encode model API documentation bindings: %w", err)
	}
	if len(raw) > ximoAIModelAPIDocsMaxStoreBytes {
		return errors.New("model API documentation bindings exceed the storage limit")
	}
	if err := s.settingRepo.Set(ctx, SettingKeyXimoAIModelAPIDocsBindings, string(raw)); err != nil {
		return fmt.Errorf("save model API documentation bindings: %w", err)
	}
	return nil
}

func ximoAIModelAPIDocsBindingKey(platform, protocol, model string) string {
	value := strings.ToLower(strings.TrimSpace(platform)) + "\x00" +
		strings.ToLower(strings.TrimSpace(protocol)) + "\x00" + strings.TrimSpace(model)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func normalizeModelAPIDocsBinding(binding ModelAPIDocsBinding) ModelAPIDocsBinding {
	binding.Platform = strings.ToLower(strings.TrimSpace(binding.Platform))
	binding.Protocol = strings.ToLower(strings.TrimSpace(binding.Protocol))
	binding.Model = strings.TrimSpace(binding.Model)
	for categoryIndex := range binding.Categories {
		category := &binding.Categories[categoryIndex]
		category.Category = strings.ToLower(strings.TrimSpace(category.Category))
		for endpointIndex := range category.Endpoints {
			endpoint := &category.Endpoints[endpointIndex]
			endpoint.Profile = strings.ToLower(strings.TrimSpace(endpoint.Profile))
			for variantIndex := range endpoint.Variants {
				endpoint.Variants[variantIndex] = strings.ToLower(strings.TrimSpace(endpoint.Variants[variantIndex]))
			}
		}
	}
	return binding
}
