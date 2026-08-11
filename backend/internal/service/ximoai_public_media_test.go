//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestXimoAIPublicHomeTabsExternalizeDataCovers(t *testing.T) {
	content := []byte("<!doctype html><style>body{color:red}</style>")
	raw := "data:text/html;base64," + base64.StdEncoding.EncodeToString(content)
	tabs := []XimoAIHomeTab{{
		ID:       "docs",
		Label:    "Docs",
		URL:      "https://docs.example",
		CoverURL: raw,
		Enabled:  true,
	}}
	repo := &ximoAIHomeTabsRepoStub{values: map[string]string{SettingKeyXimoAIHomeTabs: mustMarshalXimoAIHomeTabs(t, tabs)}}
	svc := NewSettingService(repo, &config.Config{})

	publicTabs := XimoAIPublicHomeTabs(tabs)
	require.Len(t, publicTabs, 1)
	require.NotEqual(t, raw, publicTabs[0].CoverURL)
	require.True(t, strings.HasPrefix(publicTabs[0].CoverURL, "/api/v1/settings/assets/home-tabs/docs/cover/"))
	require.Equal(t, ".html", path.Ext(publicTabs[0].CoverURL))
	require.Equal(t, raw, tabs[0].CoverURL, "public projection must not mutate admin settings")

	asset, err := svc.GetXimoAIPublicHomeCover(context.Background(), "docs", path.Base(publicTabs[0].CoverURL))
	require.NoError(t, err)
	require.Equal(t, content, asset.Content)
	require.Equal(t, "text/html", asset.ContentType)

	_, err = svc.GetXimoAIPublicHomeCover(context.Background(), "docs", "stale.html")
	require.Error(t, err)
}

func mustMarshalXimoAIHomeTabs(t *testing.T, tabs []XimoAIHomeTab) string {
	t.Helper()
	payload, err := json.Marshal(tabs)
	require.NoError(t, err)
	return string(payload)
}
