package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestXimoAIOpenAICompatibleRequestHeadersAllowed(t *testing.T) {
	for _, header := range ximoaiOpenAICompatibleRequestHeaders {
		require.True(t, openaiAllowedHeaders[header], "non-passthrough header %s should be allowed", header)
		require.True(t, openaiPassthroughAllowedHeaders[header], "passthrough header %s should be allowed", header)
		require.True(t, openaiCCRawAllowedHeaders[header], "raw chat completions header %s should be allowed", header)
	}
}

func TestOpenAIBuildUpstreamRequestPreservesXimoAICompatibleHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"gpt-5"}`)))
	c.Request.Header.Set("Idempotency-Key", "idem-123")
	c.Request.Header.Set("OpenAI-Organization", "org-123")
	c.Request.Header.Set("OpenAI-Project", "proj-123")
	c.Request.Header.Set("X-Stainless-Lang", "go")
	c.Request.Header.Set("X-Stainless-Package-Version", "1.2.3")

	svc := &OpenAIGatewayService{cfg: &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{Enabled: false},
		},
	}}
	account := &Account{
		Type:        AccountTypeAPIKey,
		Platform:    PlatformOpenAI,
		Credentials: map[string]any{"base_url": "https://example.com/v1"},
	}

	req, err := svc.buildUpstreamRequest(c.Request.Context(), c, account, []byte(`{"model":"gpt-5"}`), "token", false, "", false)
	require.NoError(t, err)
	require.Equal(t, "idem-123", req.Header.Get("Idempotency-Key"))
	require.Equal(t, "org-123", req.Header.Get("OpenAI-Organization"))
	require.Equal(t, "proj-123", req.Header.Get("OpenAI-Project"))
	require.Equal(t, "go", req.Header.Get("X-Stainless-Lang"))
	require.Equal(t, "1.2.3", req.Header.Get("X-Stainless-Package-Version"))
}
