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

func TestAccountTestService_TestAccountConnection_XimoAIOpenAICompatibleUsesOpenAIPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := Account{
		ID:          10,
		Name:        "openai-audio",
		Platform:    PlatformOpenAIAudio,
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
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
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
