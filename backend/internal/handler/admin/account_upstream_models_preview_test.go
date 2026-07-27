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
		nil,
		upstream,
		&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		nil,
		platformSvc,
	)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/models/sync-upstream-preview", handler.SyncUpstreamModelsPreview)
	return router
}

func TestAccountHandlerSyncUpstreamModelsPreview_XimoAIOpenAICompatible(t *testing.T) {
	platformSvc := service.NewPlatformService(&availableModelsPlatformRepo{platforms: map[string]service.Platform{
		service.PlatformOpenAIAudio: {
			Slug:      service.PlatformOpenAIAudio,
			Kind:      service.PlatformKindOpenAIAudio,
			Protocol:  service.PlatformProtocolOpenAICompatible,
			BaseURL:   "https://models.example.com/v1",
			AuthModes: []string{service.AccountTypeAPIKey},
			Enabled:   true,
			Builtin:   true,
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
		"platform":"openai-audio",
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
	require.Equal(t, "https://models.example.com/v1/models", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-test", upstream.lastReq.Header.Get("Authorization"))
}

func TestAccountHandlerSyncUpstreamModelsPreview_XimoAISelectableProtocols(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		kind       string
		protocol   string
		baseURL    string
		body       string
		response   string
		wantURL    string
		wantHeader string
		wantValue  string
	}{
		{
			name:       "Gemini",
			slug:       service.PlatformGrokVideo,
			kind:       service.PlatformKindGrokVideo,
			protocol:   service.PlatformProtocolGemini,
			baseURL:    "https://gemini.example.com",
			body:       `{"models":[{"name":"models/gemini-test"}]}`,
			response:   `{"platform":"grok-video","type":"apikey","protocol":"gemini","credentials":{"base_url":"https://gemini.example.com","api_key":"gemini-key"}}`,
			wantURL:    "https://gemini.example.com/v1beta/models",
			wantHeader: "x-goog-api-key",
			wantValue:  "gemini-key",
		},
		{
			name:       "Anthropic bearer",
			slug:       service.PlatformKlingAudio,
			kind:       service.PlatformKindKlingAudio,
			protocol:   service.PlatformProtocolAnthropic,
			baseURL:    "https://anthropic.example.com",
			body:       `{"data":[{"id":"claude-test"}]}`,
			response:   `{"platform":"kling_audio","type":"apikey","protocol":"gemini","credentials":{"base_url":"https://anthropic.example.com","api_key":"anthropic-key","platform_protocol":"gemini"},"extra":{"anthropic_apikey_auth_scheme":"authorization_bearer"}}`,
			wantURL:    "https://anthropic.example.com/v1/models",
			wantHeader: "Authorization",
			wantValue:  "Bearer anthropic-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platformSvc := service.NewPlatformService(&availableModelsPlatformRepo{platforms: map[string]service.Platform{
				tt.slug: {
					Slug:      tt.slug,
					Kind:      tt.kind,
					Protocol:  tt.protocol,
					BaseURL:   tt.baseURL,
					AuthModes: []string{service.AccountTypeAPIKey},
					Enabled:   true,
					Builtin:   true,
				},
			}})
			upstream := &syncUpstreamHTTPUpstream{resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}}
			router := setupSyncUpstreamModelsPreviewRouter(upstream, platformSvc)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/sync-upstream-preview", strings.NewReader(tt.response))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, tt.wantURL, upstream.lastReq.URL.String())
			require.Equal(t, tt.wantValue, upstream.lastReq.Header.Get(tt.wantHeader))
		})
	}
}
