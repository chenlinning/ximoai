package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAntigravityClaudeEmptyResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name       string
		upstream   string
		precommit  bool
		wantStatus int
		wantUsage  int
	}{
		{"single_stop", "data: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":123}}}\n\n", false, 502, 123},
		{"usage_then_stop", "data: {\"response\":{\"usageMetadata\":{\"promptTokenCount\":123}}}\n\ndata: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}]}}\n\n", false, 200, 123},
		{"stop_then_usage", "data: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}]}}\n\ndata: {\"response\":{\"usageMetadata\":{\"promptTokenCount\":123}}}\n\n", false, 502, 123},
		{"usage_only_eof", "data: {\"response\":{\"usageMetadata\":{\"promptTokenCount\":123}}}\n\n", false, 200, 123},
		{"already_committed", "data: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":123}}}\n\n", true, 200, 123},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newAntigravityTestService(&config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}})
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			if tc.precommit {
				c.Writer.WriteHeaderNow()
			}
			resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tc.upstream))}
			result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "gemini-4-hydrogen-xhigh", "gemini-4-hydrogen-xhigh")
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantUsage, result.usage.InputTokens)
			require.Equal(t, tc.wantStatus, rec.Code)
			require.Contains(t, rec.Body.String(), antigravityEmptyResponseCode)
			require.NotContains(t, rec.Body.String(), "event: message_stop")
			require.Len(t, GetOpsStreamErrors(c), 1)
			require.Equal(t, antigravityEmptyResponseCode, GetOpsStreamErrors(c)[0].Code)
		})
	}
}

func TestAntigravityClaudeNonEmptyAndNonStopResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct{ name, candidate string }{
		{"text", `{"content":{"parts":[{"text":"hello"}]},"finishReason":"STOP"}`},
		{"signed_text", `{"content":{"parts":[{"text":"hello","thoughtSignature":"sig"}]},"finishReason":"STOP"}`},
		{"thinking", `{"content":{"parts":[{"text":"reasoning","thought":true}]},"finishReason":"STOP"}`},
		{"tool", `{"content":{"parts":[{"functionCall":{"name":"test","args":{}}}]},"finishReason":"STOP"}`},
		{"image", `{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"AQ=="}}]},"finishReason":"STOP"}`},
		{"grounding", `{"groundingMetadata":{"webSearchQueries":["a"]},"finishReason":"STOP"}`},
		{"max_tokens", `{"finishReason":"MAX_TOKENS"}`},
		{"safety", `{"finishReason":"SAFETY"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newAntigravityTestService(&config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}})
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			upstream := "data: {\"response\":{\"candidates\":[" + tc.candidate + "],\"usageMetadata\":{\"promptTokenCount\":7}}}\n\n"
			resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(upstream))}
			result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "gemini-4-hydrogen-xhigh", "gemini-4-hydrogen-xhigh")
			require.NoError(t, err)
			require.Equal(t, 7, result.usage.InputTokens)
			require.Equal(t, 200, rec.Code)
			require.Contains(t, rec.Body.String(), "event: message_stop")
			require.NotContains(t, rec.Body.String(), antigravityEmptyResponseCode)
			require.Empty(t, GetOpsStreamErrors(c))
		})
	}
}

func TestAntigravityClaudeEmptyResponseForwardPreservesUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body, err := json.Marshal(map[string]any{
		"model": "hydrogen-alias", "max_tokens": 2048, "stream": true,
		"messages": []map[string]any{{"role": "user", "content": "hello"}},
	})
	require.NoError(t, err)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	upstream := "data: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":123,\"cachedContentTokenCount\":23}}}\n\n"
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   &httpUpstreamStub{resp: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(upstream))}},
	}
	account := &Account{ID: 5, Name: "test", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 1,
		Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"hydrogen-alias": "gemini-4-hydrogen-xhigh"}},
	}
	result, err := svc.Forward(context.Background(), c, account, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 100, result.Usage.InputTokens)
	require.Equal(t, 23, result.Usage.CacheReadInputTokens)
	require.Equal(t, "gemini-4-hydrogen-xhigh", result.UpstreamModel)
	require.Equal(t, 502, rec.Code)
	require.Len(t, GetOpsStreamErrors(c), 1)
}

func TestAntigravityClaudeEmptyResponseDoesNotAffectOtherModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	upstream := "data: {\"response\":{\"candidates\":[{\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":12}}}\n\n"
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(upstream))}
	result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "gemini-4-hydrogen-xhigh", "claude-sonnet-4-5")
	require.NoError(t, err)
	require.Equal(t, 12, result.usage.InputTokens)
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), "event: message_stop")
	require.NotContains(t, rec.Body.String(), antigravityEmptyResponseCode)
	require.Empty(t, GetOpsStreamErrors(c))
}
