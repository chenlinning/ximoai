package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupSyncUpstreamModelsPreviewRouter(upstream service.HTTPUpstream, platformSvc *service.PlatformService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	accountTestSvc := service.NewAccountTestService(
		nil,
		nil,
		nil,
		nil,
		upstream,
		&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		nil,
		platformSvc,
	)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/models/sync-upstream-preview", handler.SyncUpstreamModelsPreview)
	return router
}

func TestAccountHandlerSyncUpstreamModelsPreview_CustomOpenAICompatible(t *testing.T) {
	platformSvc := service.NewPlatformService(&availableModelsPlatformRepo{platforms: map[string]service.Platform{
		"mengfactory": {
			Slug:      "mengfactory",
			Protocol:  service.PlatformProtocolOpenAICompatible,
			BaseURL:   "https://models.example.com/v1",
			AuthModes: []string{service.AccountTypeAPIKey},
			Enabled:   true,
		},
	}})
	upstream := &syncUpstreamHTTPUpstream{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"vendor-audio"},{"id":"vendor-video"}]}`)),
	}}
	router := setupSyncUpstreamModelsPreviewRouter(upstream, platformSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/sync-upstream-preview", strings.NewReader(`{
		"platform":"mengfactory",
		"type":"apikey",
		"credentials":{"base_url":"https://models.example.com/v1","api_key":"sk-test"}
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Models []string `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.ElementsMatch(t, []string{"vendor-audio", "vendor-video"}, resp.Data.Models)
}
