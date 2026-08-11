//go:build unit

package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ximoAIMediaRepoStub struct {
	values map[string]string
}

func (s *ximoAIMediaRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *ximoAIMediaRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *ximoAIMediaRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *ximoAIMediaRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *ximoAIMediaRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *ximoAIMediaRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *ximoAIMediaRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestSettingHandler_GetXimoAIPublicHomeCoverServesImmutableSandboxedAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	raw := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(`<!doctype html><style>body{color:red}</style>`))
	tabsJSON, err := json.Marshal([]service.XimoAIHomeTab{{
		ID:       "docs",
		Label:    "Docs",
		URL:      "https://docs.example",
		CoverURL: raw,
		Enabled:  true,
	}})
	require.NoError(t, err)

	svc := service.NewSettingService(&ximoAIMediaRepoStub{values: map[string]string{
		service.SettingKeyXimoAIHomeTabs: string(tabsJSON),
	}}, &config.Config{})
	h := NewSettingHandler(svc, "test-version")
	publicTabs := service.XimoAIPublicHomeTabs([]service.XimoAIHomeTab{{ID: "docs", CoverURL: raw}})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{
		{Key: "id", Value: "docs"},
		{Key: "filename", Value: path.Base(publicTabs[0].CoverURL)},
	}
	c.Request = httptest.NewRequest(http.MethodGet, publicTabs[0].CoverURL, nil)

	h.GetXimoAIPublicHomeCover(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/html", recorder.Header().Get("Content-Type"))
	require.Equal(t, "public, max-age=31536000, immutable", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "same-origin", recorder.Header().Get("Cross-Origin-Resource-Policy"))
	require.Contains(t, recorder.Header().Get("Content-Security-Policy"), "sandbox")
	require.NotEmpty(t, recorder.Header().Get("ETag"))
}
