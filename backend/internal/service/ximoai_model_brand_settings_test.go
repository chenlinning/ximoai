package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXimoAIModelBrandSettingsSeparatesSameModelByPlatform(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	first := ModelBrandOverride{Platform: "custom-a", Model: "same-model", Brand: "Brand A"}
	second := ModelBrandOverride{Platform: "custom-b", Model: "same-model", Brand: "Brand B"}

	require.NoError(t, svc.SaveXimoAIModelBrandOverride(context.Background(), first))
	require.NoError(t, svc.SaveXimoAIModelBrandOverride(context.Background(), second))

	got, err := svc.GetXimoAIModelBrandOverrides(context.Background(), []ModelBrandOverride{
		{Platform: first.Platform, Model: first.Model},
		{Platform: second.Platform, Model: second.Model},
	})
	require.NoError(t, err)
	require.Equal(t, first, *got[0])
	require.Equal(t, second, *got[1])
}

func TestXimoAIModelBrandSettingsDeleteRestoresOnlyTarget(t *testing.T) {
	repo := &modelAPIDocsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)
	first := ModelBrandOverride{Platform: "custom", Model: "model-a", Brand: "Brand A"}
	second := ModelBrandOverride{Platform: "custom", Model: "model-b", Brand: "Brand B"}
	require.NoError(t, svc.SaveXimoAIModelBrandOverride(context.Background(), first))
	require.NoError(t, svc.SaveXimoAIModelBrandOverride(context.Background(), second))

	require.NoError(t, svc.DeleteXimoAIModelBrandOverride(context.Background(), first.Platform, first.Model))
	got, err := svc.GetXimoAIModelBrandOverrides(context.Background(), []ModelBrandOverride{first, second})
	require.NoError(t, err)
	require.Nil(t, got[0])
	require.Equal(t, second, *got[1])
}

func TestValidateXimoAIModelBrand(t *testing.T) {
	require.NoError(t, ValidateXimoAIModelBrand("Moonshot AI"))
	require.Error(t, ValidateXimoAIModelBrand(""))
	require.Error(t, ValidateXimoAIModelBrand("bad\nbrand"))
	require.Error(t, ValidateXimoAIModelBrand(string(make([]rune, 65))))
}
