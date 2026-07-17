package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestForwardGrokMediaUsesSub2APIProtocolForXimoAIGrokVideoGeneration(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"grok-video-3-10s",
		"prompt":"waves",
		"resolution":"720p",
		"duration":8,
		"image":{"image_url":"https://example.com/input.png"}
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	account := newXimoAIGrokVideoMediaTestAccount()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"task_123","status":"pending"}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream, cfg: &config.Config{}}

	result, err := svc.ForwardGrokMedia(context.Background(), c, account, GrokMediaEndpointVideosGenerations, "", body, "application/json")

	require.NoError(t, err)
	require.Equal(t, "https://video.test/v1/video/create", upstream.lastReq.URL.String())
	require.JSONEq(t, `{
		"model":"grok-video-3-10s",
		"prompt":"waves",
		"aspect_ratio":"3:2",
		"size":"720P",
		"images":["https://example.com/input.png"]
	}`, string(upstream.lastBody))
	require.JSONEq(t, `{"id":"task_123","request_id":"task_123","status":"pending"}`, recorder.Body.String())
	require.Equal(t, "task_123", result.ResponseID)
	require.Equal(t, "grok-video-3-10s", result.Model)
	require.Equal(t, 10, result.VideoDurationSeconds)
}

func TestForwardGrokMediaPreservesSub2APIStatusForXimoAIGrokVideo(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_123", nil)
	responseBody := `{"id":"task_123","status":"completed","progress":100,"video_url":"https://example.com/result.mp4"}`

	account := newXimoAIGrokVideoMediaTestAccount()
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream, cfg: &config.Config{}}

	_, err := svc.ForwardGrokMedia(context.Background(), c, account, GrokMediaEndpointVideoStatus, "task_123", nil, "")

	require.NoError(t, err)
	require.Equal(t, "https://video.test/v1/video/query?id=task_123", upstream.lastReq.URL.String())
	require.JSONEq(t, responseBody, recorder.Body.String())
}

func TestForwardGrokMediaUsesSub2APITransportFailoverForXimoAIGrokVideo(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"grok-video-3-10s","prompt":"waves"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{err: errors.New("connection reset")}
	svc := &OpenAIGatewayService{httpUpstream: upstream, cfg: &config.Config{}}

	_, err := svc.ForwardGrokMedia(context.Background(), c, newXimoAIGrokVideoMediaTestAccount(), GrokMediaEndpointVideosGenerations, "", body, "application/json")

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
}

func newXimoAIGrokVideoMediaTestAccount() *Account {
	return &Account{
		ID:          321,
		Name:        "grok-video",
		Platform:    PlatformGrokVideo,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "test-key",
			"base_url": "https://video.test/v1",
		},
	}
}
