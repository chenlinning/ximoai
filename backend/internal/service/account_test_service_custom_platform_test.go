package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAccountTestService_TestAccountConnection_CustomOpenAICompatibleUsesOpenAIPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := Account{
		ID:          10,
		Name:        "acme-openai",
		Platform:    "acme-openai",
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-acme",
			"base_url": "https://api.acme.test/v1",
		},
	}
	repo := stubOpenAIAccountRepo{accounts: []Account{account}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\ndata: {\"type\":\"response.completed\"}\n\n")),
	}}
	platformService := NewPlatformService(&platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			"acme-openai": {
				Slug:        "acme-openai",
				DisplayName: "Acme OpenAI",
				Protocol:    PlatformProtocolOpenAICompatible,
				BaseURL:     "https://api.acme.test/v1",
				AuthModes:   []string{AccountTypeAPIKey},
				Enabled:     true,
			},
		},
	})
	svc := &AccountTestService{
		accountRepo:     repo,
		httpUpstream:    upstream,
		cfg:             &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		platformService: platformService,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/10/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-4.1", "", AccountTestModeDefault)
	require.NoError(t, err)

	require.Equal(t, "https://api.acme.test/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-acme", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "gpt-4.1", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
}

func TestAccountTestService_TestAccountConnection_CustomGeminiCompatibleUsesGeminiPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := Account{
		ID:          11,
		Name:        "acme-gemini",
		Platform:    "acme-gemini",
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "gemini-key",
			"base_url": "https://gemini.acme.test",
		},
	}
	repo := stubOpenAIAccountRepo{accounts: []Account{account}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}]}\n\n")),
	}}
	platformService := NewPlatformService(&platformRepoStubForPlatformService{
		platforms: map[string]Platform{
			"acme-gemini": {
				Slug:        "acme-gemini",
				DisplayName: "Acme Gemini",
				Protocol:    PlatformProtocolGemini,
				BaseURL:     "https://gemini.acme.test",
				AuthModes:   []string{AccountTypeAPIKey},
				Enabled:     true,
			},
		},
	})
	svc := &AccountTestService{
		accountRepo:     repo,
		httpUpstream:    upstream,
		cfg:             &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		platformService: platformService,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/11/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gemini-2.5-pro", "", AccountTestModeDefault)
	require.NoError(t, err)

	require.Equal(t, "https://gemini.acme.test/v1beta/models/gemini-2.5-pro:streamGenerateContent?alt=sse", upstream.lastReq.URL.String())
	require.Equal(t, "gemini-key", upstream.lastReq.Header.Get("x-goog-api-key"))
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
}
