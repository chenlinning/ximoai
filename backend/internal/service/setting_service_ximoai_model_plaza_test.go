//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSettingService_XimoAIModelPlazaEntryDefaultsEnabled(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

	require.True(t, svc.parseSettings(map[string]string{}).XimoAIModelPlazaEntryEnabled)

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.XimoAIModelPlazaEntryEnabled)
}

func TestSettingService_XimoAIModelPlazaEntryCanBeDisabled(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyXimoAIModelPlazaEntryEnabled: "false",
	}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.XimoAIModelPlazaEntryEnabled)

	repo := &settingUpdateRepoStub{}
	updateService := NewSettingService(repo, &config.Config{})
	require.NoError(t, updateService.UpdateSettings(context.Background(), &SystemSettings{
		XimoAIModelPlazaEntryEnabled: false,
	}))
	require.Equal(t, "false", repo.updates[SettingKeyXimoAIModelPlazaEntryEnabled])
}
