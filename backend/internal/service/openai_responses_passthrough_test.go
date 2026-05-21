package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIResponsesPassthroughID(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/v1/responses/resp_123", want: "resp_123"},
		{path: "/v1/responses/resp_123/cancel", want: "resp_123"},
		{path: "/responses/resp_456/input_items", want: "resp_456"},
		{path: "/v1/responses/", want: ""},
		{path: "/chat/completions", want: ""},
	}

	for _, tt := range tests {
		require.Equal(t, tt.want, ExtractOpenAIResponsesPassthroughID(tt.path), "path=%s", tt.path)
	}
}

func TestOpenAIGatewayService_ForwardOpenAIResponsesPassthroughGET(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/responses/resp_123/input_items?after=item_1", nil)
	c.Request.Header.Set("Accept", "application/json")
	c.Request.Header.Set("Authorization", "Bearer client-key")
	c.Request.Header.Set("X-Test", "drop")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"rid_resp_get"},
		},
		Body: io.NopCloser(strings.NewReader(`{"object":"list","data":[]}`)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          88,
		Name:        "custom-openai",
		Platform:    "acme",
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":    "sk-custom",
			"base_url":   "https://api.acme.test/v1",
			"user_agent": "custom-agent",
		},
	}

	err := svc.ForwardOpenAIResponsesPassthrough(context.Background(), c, account, http.MethodGet, nil)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, http.MethodGet, upstream.lastReq.Method)
	require.Equal(t, "https://api.acme.test/v1/responses/resp_123/input_items?after=item_1", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-custom", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "custom-agent", upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "application/json", upstream.lastReq.Header.Get("Accept"))
	require.Empty(t, upstream.lastReq.Header.Get("X-Test"))
	require.Nil(t, upstream.lastReq.Body)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"object":"list","data":[]}`, rec.Body.String())
}

func TestOpenAIGatewayService_ForwardOpenAIResponsesPassthroughPOSTCancel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"reason":"user"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/resp_123/cancel", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_123","status":"cancelled"}`)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          89,
		Name:        "custom-openai",
		Platform:    "acme",
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-custom",
			"base_url": "https://api.acme.test",
		},
	}

	err := svc.ForwardOpenAIResponsesPassthrough(context.Background(), c, account, http.MethodPost, body)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, http.MethodPost, upstream.lastReq.Method)
	require.Equal(t, "https://api.acme.test/v1/responses/resp_123/cancel", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-custom", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "application/json", upstream.lastReq.Header.Get("Content-Type"))
	require.Equal(t, body, upstream.lastBody)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"id":"resp_123","status":"cancelled"}`, rec.Body.String())
}
