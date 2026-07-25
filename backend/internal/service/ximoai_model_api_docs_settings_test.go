package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type modelAPIDocsSettingRepoStub struct {
	values map[string]string
}

func (s *modelAPIDocsSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *modelAPIDocsSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *modelAPIDocsSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *modelAPIDocsSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *modelAPIDocsSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

func (s *modelAPIDocsSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *modelAPIDocsSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestXimoAIModelAPIDocsSettingsRoundTripPreservesMultipleVariants(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	binding := ModelAPIDocsBinding{
		Platform: "volcengine-agent-plan",
		Protocol: "native",
		Model:    "seed-tts-2.0",
		Categories: []ModelAPIDocsCategoryBinding{{
			Category: "tts",
			Endpoints: []ModelAPIDocsEndpointBinding{
				{Profile: "volcengine_tts_unidirectional", Variants: []string{"sync"}},
				{Profile: "volcengine_tts_unidirectional_stream", Variants: []string{"stream"}},
				{Profile: "volcengine_tts_bidirectional", Variants: []string{"bidirectional"}},
			},
		}},
	}

	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), binding))
	got, err := svc.GetXimoAIModelAPIDocsBinding(context.Background(), binding.Platform, binding.Protocol, binding.Model)
	require.NoError(t, err)
	require.Equal(t, binding, *got)
}

func TestXimoAIModelAPIDocsSettingsKeyIncludesProtocol(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	openAI := ModelAPIDocsBinding{Platform: "custom", Protocol: "openai_compatible", Model: "same-model"}
	anthropic := ModelAPIDocsBinding{Platform: "custom", Protocol: "anthropic", Model: "same-model"}

	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), openAI))
	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), anthropic))

	gotOpenAI, err := svc.GetXimoAIModelAPIDocsBinding(context.Background(), openAI.Platform, openAI.Protocol, openAI.Model)
	require.NoError(t, err)
	gotAnthropic, err := svc.GetXimoAIModelAPIDocsBinding(context.Background(), anthropic.Platform, anthropic.Protocol, anthropic.Model)
	require.NoError(t, err)
	require.Equal(t, "openai_compatible", gotOpenAI.Protocol)
	require.Equal(t, "anthropic", gotAnthropic.Protocol)
}

func TestXimoAIModelAPIDocsSettingsBatchLookupPreservesTargetOrder(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	first := ModelAPIDocsBinding{Platform: "custom", Protocol: "openai_compatible", Model: "model-a"}
	second := ModelAPIDocsBinding{Platform: "custom", Protocol: "anthropic", Model: "model-b"}
	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), first))
	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), second))

	got, err := svc.GetXimoAIModelAPIDocsBindings(context.Background(), []ModelAPIDocsBinding{
		first,
		{Platform: "custom", Protocol: "openai_compatible", Model: "missing"},
		second,
	})
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, first, *got[0])
	require.Nil(t, got[1])
	require.Equal(t, second, *got[2])
}

func TestXimoAIModelAPIDocsSettingsDeleteOnlyExactBinding(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	first := ModelAPIDocsBinding{Platform: "custom", Protocol: "openai_compatible", Model: "model-a"}
	second := ModelAPIDocsBinding{Platform: "custom", Protocol: "openai_compatible", Model: "model-b"}
	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), first))
	require.NoError(t, svc.SaveXimoAIModelAPIDocsBinding(context.Background(), second))

	require.NoError(t, svc.DeleteXimoAIModelAPIDocsBinding(context.Background(), first.Platform, first.Protocol, first.Model))
	gotFirst, err := svc.GetXimoAIModelAPIDocsBinding(context.Background(), first.Platform, first.Protocol, first.Model)
	require.NoError(t, err)
	require.Nil(t, gotFirst)
	gotSecond, err := svc.GetXimoAIModelAPIDocsBinding(context.Background(), second.Platform, second.Protocol, second.Model)
	require.NoError(t, err)
	require.Equal(t, second, *gotSecond)
}

func TestXimoAIModelAPIDocsSettingsRejectsMalformedStore(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{
		SettingKeyXimoAIModelAPIDocsBindings: `{not-json`,
	}}
	svc := NewSettingService(repo, nil)

	_, err := svc.GetXimoAIModelAPIDocsBinding(context.Background(), "custom", "openai_compatible", "model")
	require.Error(t, err)
}
