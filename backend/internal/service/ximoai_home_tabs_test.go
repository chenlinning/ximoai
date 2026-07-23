//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type ximoAIHomeTabsRepoStub struct {
	values map[string]string
}

func (s *ximoAIHomeTabsRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *ximoAIHomeTabsRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *ximoAIHomeTabsRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *ximoAIHomeTabsRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *ximoAIHomeTabsRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *ximoAIHomeTabsRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *ximoAIHomeTabsRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestXimoAIHomeTabsMissingSettingReturnsEmptyList(t *testing.T) {
	svc := NewSettingService(&ximoAIHomeTabsRepoStub{values: map[string]string{}}, &config.Config{})

	tabs, err := svc.GetXimoAIHomeTabs(context.Background())

	require.NoError(t, err)
	require.Empty(t, tabs)
}

func TestUpdateXimoAIHomeTabsNormalizesAndPersists(t *testing.T) {
	repo := &ximoAIHomeTabsRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})
	invalidated := 0
	svc.SetOnUpdateCallback(func() { invalidated++ })

	tabs, err := svc.UpdateXimoAIHomeTabs(context.Background(), []XimoAIHomeTab{
		{
			Label:        "  Workbench  ",
			URL:          "https://workbench.ximoai.cn/",
			CoverURL:     " /images/workbench.webp ",
			Enabled:      true,
			WorkbenchSSO: true,
			SortOrder:    99,
		},
		{
			ID:        "docs",
			Label:     " Docs ",
			URL:       "https://docs.ximoai.cn/start",
			Enabled:   true,
			SortOrder: -10,
		},
	})

	require.NoError(t, err)
	require.Len(t, tabs, 2)
	require.Regexp(t, `^[a-f0-9]{16}$`, tabs[0].ID)
	require.Equal(t, "Workbench", tabs[0].Label)
	require.Equal(t, "/images/workbench.webp", tabs[0].CoverURL)
	require.Equal(t, 0, tabs[0].SortOrder)
	require.Equal(t, "docs", tabs[1].ID)
	require.Equal(t, 1, tabs[1].SortOrder)
	require.Equal(t, 1, invalidated)
	require.NotEmpty(t, repo.values[SettingKeyXimoAIHomeTabs])

	roundTrip, err := svc.GetXimoAIHomeTabs(context.Background())
	require.NoError(t, err)
	require.Equal(t, tabs, roundTrip)
}

func TestUpdateXimoAIHomeTabsCanClearAllTabs(t *testing.T) {
	repo := &ximoAIHomeTabsRepoStub{values: map[string]string{SettingKeyXimoAIHomeTabs: `[{"id":"docs"}]`}}
	svc := NewSettingService(repo, &config.Config{})

	tabs, err := svc.UpdateXimoAIHomeTabs(context.Background(), []XimoAIHomeTab{})

	require.NoError(t, err)
	require.Empty(t, tabs)
	require.Equal(t, "[]", repo.values[SettingKeyXimoAIHomeTabs])
}

func TestUpdateXimoAIHomeTabsRejectsUnsafeAndDuplicateValues(t *testing.T) {
	tests := []struct {
		name string
		tabs []XimoAIHomeTab
	}{
		{
			name: "javascript URL",
			tabs: []XimoAIHomeTab{{Label: "Bad", URL: "javascript:alert(1)", Enabled: true}},
		},
		{
			name: "URL credentials",
			tabs: []XimoAIHomeTab{{Label: "Bad", URL: "https://user:pass@example.com", Enabled: true}},
		},
		{
			name: "unsafe cover URL",
			tabs: []XimoAIHomeTab{{Label: "Bad", URL: "https://example.com", CoverURL: "data:text/html,bad", Enabled: true}},
		},
		{
			name: "duplicate ID",
			tabs: []XimoAIHomeTab{
				{ID: "same", Label: "One", URL: "https://one.example", Enabled: true},
				{ID: "same", Label: "Two", URL: "https://two.example", Enabled: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSettingService(&ximoAIHomeTabsRepoStub{values: map[string]string{}}, &config.Config{})

			_, err := svc.UpdateXimoAIHomeTabs(context.Background(), tt.tabs)

			require.Error(t, err)
		})
	}
}

func TestXimoAIHomeTabFrameOriginsAreDeduplicated(t *testing.T) {
	tabs := []XimoAIHomeTab{
		{URL: "https://workbench.ximoai.cn/path", Enabled: true},
		{URL: "https://workbench.ximoai.cn/other", Enabled: true},
		{URL: "http://localhost:4173/page", Enabled: true},
		{URL: "https://disabled.example", Enabled: false},
	}

	require.Equal(t, []string{
		"https://workbench.ximoai.cn",
		"http://localhost:4173",
	}, XimoAIHomeTabFrameOrigins(tabs))
}
