package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type modelMetadataSettingRepoStub struct {
	values map[string]string
}

func (s *modelMetadataSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *modelMetadataSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *modelMetadataSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *modelMetadataSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *modelMetadataSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

func (s *modelMetadataSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *modelMetadataSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestXimoAIModelMetadataSettingsPreservePartialOverrides(t *testing.T) {
	repo := &modelMetadataSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	brand := "XimoAI Lab"
	types := []string{"conversation", "image"}
	override := ModelMetadataOverride{
		Platform: "custom-openai",
		Model:    "same-model",
		Brand:    &brand,
		Types:    &types,
	}

	require.NoError(t, svc.SaveXimoAIModelMetadataOverride(context.Background(), override))
	got, err := svc.GetXimoAIModelMetadataOverrides(context.Background(), []ModelMetadataOverride{{
		Platform: override.Platform,
		Model:    override.Model,
	}})
	require.NoError(t, err)
	require.Equal(t, override, *got[0])
	require.Nil(t, got[0].InvocationModes)
}

func TestXimoAIModelMetadataSettingsLoadsLegacyBrandOverrides(t *testing.T) {
	legacy := map[string]any{
		"version": 1,
		"overrides": map[string]any{
			ximoAIModelMetadataOverrideKey("custom-openai", "same-model"): map[string]any{
				"platform": "custom-openai",
				"model":    "same-model",
				"brand":    "Legacy Brand",
			},
		},
	}
	raw, err := json.Marshal(legacy)
	require.NoError(t, err)
	repo := &modelMetadataSettingRepoStub{values: map[string]string{
		SettingKeyXimoAIModelBrandOverrides: string(raw),
	}}
	svc := NewSettingService(repo, nil)

	got, err := svc.GetXimoAIModelMetadataOverrides(context.Background(), []ModelMetadataOverride{{
		Platform: "custom-openai",
		Model:    "same-model",
	}})
	require.NoError(t, err)
	require.NotNil(t, got[0])
	require.Equal(t, "Legacy Brand", *got[0].Brand)
	require.Nil(t, got[0].Types)
	require.Nil(t, got[0].InvocationModes)
}

func TestXimoAIModelMetadataSettingsPersistsReasoningAndThinkingOverrides(t *testing.T) {
	repo := &modelMetadataSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	levels := []string{"low", "high"}
	thinking := true
	override := ModelMetadataOverride{
		Platform:          "custom-openai",
		Model:             "mapped-gpt",
		ReasoningLevels:   &levels,
		ThinkingSupported: &thinking,
	}

	if err := svc.SaveXimoAIModelMetadataOverride(context.Background(), override); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetXimoAIModelMetadataOverrides(context.Background(), []ModelMetadataOverride{{Platform: override.Platform, Model: override.Model}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] == nil {
		t.Fatal("reasoning override was not persisted")
	}
	if got[0].ThinkingSupported == nil || !*got[0].ThinkingSupported {
		t.Fatal("thinking override was not persisted")
	}
	if len(*got[0].ReasoningLevels) != 2 || (*got[0].ReasoningLevels)[1] != "high" {
		t.Fatalf("unexpected reasoning levels: %#v", got[0].ReasoningLevels)
	}
}

func TestXimoAIModelMetadataSettingsDeleteOnlyExactTarget(t *testing.T) {
	repo := &modelMetadataSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	firstBrand := "Brand A"
	secondBrand := "Brand B"
	first := ModelMetadataOverride{Platform: "custom", Model: "model-a", Brand: &firstBrand}
	second := ModelMetadataOverride{Platform: "custom", Model: "model-b", Brand: &secondBrand}
	require.NoError(t, svc.SaveXimoAIModelMetadataOverride(context.Background(), first))
	require.NoError(t, svc.SaveXimoAIModelMetadataOverride(context.Background(), second))

	require.NoError(t, svc.DeleteXimoAIModelMetadataOverride(context.Background(), first.Platform, first.Model))
	got, err := svc.GetXimoAIModelMetadataOverrides(context.Background(), []ModelMetadataOverride{first, second})
	require.NoError(t, err)
	require.Nil(t, got[0])
	require.Equal(t, second, *got[1])
}
