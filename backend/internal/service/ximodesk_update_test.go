package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type ximoDeskSettingRepoStub struct {
	values map[string]string
}

func newXimoDeskSettingRepoStub() *ximoDeskSettingRepoStub {
	return &ximoDeskSettingRepoStub{values: map[string]string{}}
}

func (r *ximoDeskSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (r *ximoDeskSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (r *ximoDeskSettingRepoStub) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *ximoDeskSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *ximoDeskSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}

func (r *ximoDeskSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *ximoDeskSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func newXimoDeskTestSettingService() (*SettingService, *ximoDeskSettingRepoStub) {
	repo := newXimoDeskSettingRepoStub()
	return NewSettingService(repo, &config.Config{}), repo
}

func validXimoDeskUpdateConfig() *XimoDeskUpdateConfig {
	return &XimoDeskUpdateConfig{
		Enabled: true,
		Releases: []XimoDeskUpdateRelease{
			{
				Channel:     "stable",
				OS:          "windows",
				Arch:        "x86_64",
				Version:     "1.0.1",
				DownloadURL: "https://www.ximoai.cn/downloads/XimoDesk-1.0.1-x64.msi",
				Notes:       "fixes",
				SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Force:       false,
			},
		},
	}
}

func TestXimoDeskUpdateConfigSaveAndResolve(t *testing.T) {
	svc, _ := newXimoDeskTestSettingService()
	ctx := context.Background()

	require.NoError(t, svc.SaveXimoDeskUpdateConfig(ctx, validXimoDeskUpdateConfig()))

	update, ok, err := svc.ResolveXimoAppUpdate(ctx, "ximodesk", XimoDeskVersionRequest{
		App:     "XimoDesk",
		Type:    "ximodesk-client",
		OS:      "windows",
		Arch:    "amd64",
		Channel: "stable",
	})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "1.0.1", update.Version)
	require.Equal(t, "https://www.ximoai.cn/downloads/XimoDesk-1.0.1-x64.msi", update.DownloadURL)
}

func TestXimoDeskUpdateResolveNoMatch(t *testing.T) {
	svc, _ := newXimoDeskTestSettingService()
	ctx := context.Background()
	require.NoError(t, svc.SaveXimoDeskUpdateConfig(ctx, validXimoDeskUpdateConfig()))

	_, ok, err := svc.ResolveXimoAppUpdate(ctx, "ximodesk", XimoDeskVersionRequest{
		App:     "XimoDesk",
		Type:    "ximodesk-client",
		OS:      "windows",
		Arch:    "aarch64",
		Channel: "stable",
	})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestXimoDeskUpdateConfigValidation(t *testing.T) {
	svc, _ := newXimoDeskTestSettingService()
	ctx := context.Background()

	cfg := validXimoDeskUpdateConfig()
	cfg.Releases[0].SHA256 = "bad"
	require.Error(t, svc.SaveXimoDeskUpdateConfig(ctx, cfg))

	cfg = validXimoDeskUpdateConfig()
	cfg.Releases[0].DownloadURL = "http://www.ximoai.cn/downloads/XimoDesk-1.0.1-x64.msi"
	require.Error(t, svc.SaveXimoDeskUpdateConfig(ctx, cfg))

	cfg = validXimoDeskUpdateConfig()
	cfg.Releases[0].DownloadURL = "https://example.com/XimoDesk-1.0.1-x64.msi"
	require.Error(t, svc.SaveXimoDeskUpdateConfig(ctx, cfg))
}

func TestXimoDeskPackageUploadStoresFileAndResolvesLocale(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(ximoAppPackageDirEnv, dir)
	t.Setenv(ximoAppDownloadBaseURLEnv, "https://ximoai.cn/downloads/ximoapp")
	svc, _ := newXimoDeskTestSettingService()
	ctx := context.Background()

	release, cfg, err := svc.SaveXimoDeskPackageRelease(ctx, XimoDeskPackageUpload{
		Channel:      "stable",
		OS:           "windows",
		Arch:         "x64",
		Locale:       "zh-CN",
		Version:      "1.0.2",
		OriginalName: "installer.msi",
		Notes:        "zh build",
		Enabled:      true,
		Reader:       strings.NewReader("ximodesk-package"),
	})
	require.NoError(t, err)
	require.NotNil(t, release)
	require.True(t, cfg.Enabled)
	require.Equal(t, "zh-cn", release.Locale)
	require.Equal(t, "msi", release.PackageType)
	require.Equal(t, "1.0.2", release.Version)
	require.Equal(t, "e5273b49adc1ccfc37836fa03584a811d957e58dca6d9c791fbcaddbd07fe159", release.SHA256)
	require.FileExists(t, filepath.Join(dir, release.FileName))
	require.Contains(t, release.DownloadURL, "/downloads/ximoapp/")

	update, ok, err := svc.ResolveXimoAppUpdate(ctx, "ximodesk", XimoDeskVersionRequest{
		App:     "XimoDesk",
		Type:    "ximodesk-client",
		OS:      "windows",
		Arch:    "x86_64",
		Channel: "stable",
		Locale:  "zh-CN",
	})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "1.0.2", update.Version)

	savedPath := filepath.Join(dir, release.FileName)
	cfg, err = svc.DeleteXimoDeskUpdateRelease(ctx, release.ID)
	require.NoError(t, err)
	require.Empty(t, cfg.Releases)
	_, err = os.Stat(savedPath)
	require.True(t, os.IsNotExist(err))
}

func TestXimoAppPackageUploadStoresMobileReleaseAndResolvesGenericEndpoint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(ximoAppPackageDirEnv, dir)
	t.Setenv(ximoAppDownloadBaseURLEnv, "https://ximoai.cn/downloads/ximoapp")
	svc, _ := newXimoDeskTestSettingService()
	ctx := context.Background()

	release, cfg, err := svc.SaveXimoDeskPackageRelease(ctx, XimoDeskPackageUpload{
		AppKey:                  "ximo-mobile",
		Channel:                 "stable",
		OS:                      "android",
		Arch:                    "universal",
		Locale:                  "all",
		Version:                 "2.0.0",
		VersionCode:             200,
		MinSupportedVersion:     "1.5.0",
		MinSupportedVersionCode: 150,
		OriginalName:            "ximomobile.apk",
		Notes:                   "android build",
		Enabled:                 true,
		Reader:                  strings.NewReader("ximo-mobile-apk"),
	})
	require.NoError(t, err)
	require.NotNil(t, release)
	require.True(t, cfg.Enabled)
	require.Equal(t, "ximo-mobile", release.AppKey)
	require.Equal(t, "android", release.OS)
	require.Equal(t, "universal", release.Arch)
	require.Equal(t, "apk", release.PackageType)
	require.EqualValues(t, 200, release.VersionCode)
	require.Equal(t, "1.5.0", release.MinSupportedVersion)
	require.EqualValues(t, 150, release.MinSupportedVersionCode)
	require.Contains(t, release.DownloadURL, "/downloads/ximoapp/")
	require.FileExists(t, filepath.Join(dir, release.FileName))

	update, ok, err := svc.ResolveXimoAppUpdate(ctx, "ximo-mobile", XimoDeskVersionRequest{
		AppKey:      "ximo-mobile",
		Version:     "1.0.0",
		VersionCode: 100,
		Platform:    "android",
		Arch:        "arm64",
		Channel:     "stable",
		Locale:      "zh-CN",
	})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "ximo-mobile", update.AppKey)
	require.Equal(t, "2.0.0", update.Version)
	require.EqualValues(t, 200, update.VersionCode)
	require.Equal(t, "1.5.0", update.MinSupportedVersion)
	require.EqualValues(t, 150, update.MinSupportedVersionCode)
}

func TestXimoAppDownloadCenterReturnsEnabledAppsAndReleases(t *testing.T) {
	svc, _ := newXimoDeskTestSettingService()
	ctx := context.Background()
	enabled := true
	disabled := false

	require.NoError(t, svc.SaveXimoDeskUpdateConfig(ctx, &XimoDeskUpdateConfig{
		Enabled: true,
		Apps: []XimoAppUpdateApp{
			{
				Key:         "ximodesk",
				Name:        "XimoDesk",
				Description: "Desktop client",
				ClientType:  "desktop",
				Enabled:     &enabled,
			},
			{
				Key:         "disabled-app",
				Name:        "Disabled",
				Description: "Should not be visible",
				ClientType:  "desktop",
				Enabled:     &disabled,
			},
		},
		Releases: []XimoDeskUpdateRelease{
			{
				AppKey:      "ximodesk",
				Enabled:     &enabled,
				Channel:     "stable",
				OS:          "windows",
				Arch:        "x86_64",
				Locale:      "zh-CN",
				PackageType: "msi",
				Version:     "1.0.2",
				DownloadURL: "https://www.ximoai.cn/downloads/ximoapp/XimoAPP-ximodesk-1.0.2-windows-x86_64-zh-cn.msi",
				Notes:       "newer build",
				SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				FileName:    "XimoAPP-ximodesk-1.0.2-windows-x86_64-zh-cn.msi",
			},
			{
				AppKey:      "ximodesk",
				Enabled:     &disabled,
				Channel:     "stable",
				OS:          "windows",
				Arch:        "x86_64",
				Locale:      "zh-CN",
				PackageType: "msi",
				Version:     "1.0.1",
				DownloadURL: "https://www.ximoai.cn/downloads/ximoapp/disabled.msi",
				Notes:       "disabled build",
				SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			{
				AppKey:      "disabled-app",
				Enabled:     &enabled,
				Channel:     "stable",
				OS:          "windows",
				Arch:        "x86_64",
				Locale:      "zh-CN",
				PackageType: "msi",
				Version:     "1.0.0",
				DownloadURL: "https://www.ximoai.cn/downloads/ximoapp/hidden.msi",
				Notes:       "hidden app",
				SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
		},
	}))

	center, err := svc.GetXimoAppDownloadCenter(ctx)
	require.NoError(t, err)
	require.Len(t, center.Apps, 1)
	require.Equal(t, "ximodesk", center.Apps[0].Key)
	require.Equal(t, "Desktop client", center.Apps[0].Description)
	require.Len(t, center.Apps[0].Releases, 1)
	require.Equal(t, "1.0.2", center.Apps[0].Releases[0].Version)
	require.Equal(t, "newer build", center.Apps[0].Releases[0].Notes)
}
