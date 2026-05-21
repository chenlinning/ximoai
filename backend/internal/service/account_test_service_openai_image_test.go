package service

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAccountTestService_OpenAIImageOAuthHandlesOutputItemDoneFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
			},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_123\",\"type\":\"image_generation_call\",\"result\":\"aGVsbG8=\",\"revised_prompt\":\"draw a cat\",\"output_format\":\"png\"}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"created_at\":1710000006,\"tool_usage\":{\"image_gen\":{\"images\":1}},\"output\":[]}}\n\n" +
					"data: [DONE]\n\n",
			)),
		},
	}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:       53,
		Name:     "openai-oauth",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token-123",
		},
	}

	err := svc.testOpenAIImageOAuth(c, context.Background(), account, "gpt-image-2", "draw a cat")
	require.NoError(t, err)
	require.Contains(t, rec.Body.String(), "Calling Codex /responses image tool")
	require.Contains(t, rec.Body.String(), "data:image/png;base64,aGVsbG8=")
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestAccountTestService_OpenAIImageAPIKeyUsesConfiguredV1BaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"aGVsbG8=","revised_prompt":"draw a cat"}]}`)),
		},
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       54,
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://image-upstream.example/v1",
		},
	}

	err := svc.testOpenAIImageAPIKey(c, context.Background(), account, "gpt-image-2", "draw a cat")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://image-upstream.example/v1/images/generations", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer test-api-key", upstream.lastReq.Header.Get("Authorization"))
	require.Contains(t, rec.Body.String(), "data:image/png;base64,aGVsbG8=")
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestAccountTestService_OpenAIAudioModelUsesAudioSpeechEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"audio/mpeg"},
			},
			Body: io.NopCloser(strings.NewReader("mp3-bytes")),
		},
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       55,
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://audio-upstream.example/v1",
		},
	}

	err := svc.testOpenAIAccountConnection(c, account, "gpt-4o-audio-preview", "", "")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://audio-upstream.example/v1/audio/speech", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer test-api-key", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "audio/mpeg", upstream.lastReq.Header.Get("Accept"))
	require.Equal(t, "gpt-4o-audio-preview", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "hi", gjson.GetBytes(upstream.lastBody, "input").String())
	require.Equal(t, "alloy", gjson.GetBytes(upstream.lastBody, "voice").String())
	require.Contains(t, rec.Body.String(), "Audio test returned")
	require.Contains(t, rec.Body.String(), `"type":"audio"`)
	require.Contains(t, rec.Body.String(), `data:audio/mpeg;base64,bXAzLWJ5dGVz`)
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestAccountTestService_OpenAIVideoModelUsesVideosEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

	upstream := &httpUpstreamRecorder{
		responses: []*http.Response{
			{
				StatusCode: http.StatusAccepted,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(strings.NewReader(`{"id":"video_123","status":"queued"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(strings.NewReader(`{"id":"video_123","status":"completed"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"video/mp4"},
				},
				Body: io.NopCloser(strings.NewReader("mp4-bytes")),
			},
		},
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       56,
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://video-upstream.example/v1",
		},
	}

	err := svc.testOpenAIAccountConnection(c, account, "sora-2", "", "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 3)
	createReq := upstream.requests[0]
	require.NotNil(t, createReq)
	require.Equal(t, "https://video-upstream.example/v1/videos", createReq.URL.String())
	require.Equal(t, "Bearer test-api-key", createReq.Header.Get("Authorization"))
	require.Equal(t, "application/json", createReq.Header.Get("Accept"))
	form := parseMultipartFormValues(t, createReq.Header.Get("Content-Type"), upstream.bodies[0])
	require.Equal(t, "sora-2", form["model"])
	require.Equal(t, "A tiny test video of a sunrise over mountains.", form["prompt"])
	require.Equal(t, "4", form["seconds"])
	require.Equal(t, "720x1280", form["size"])
	require.Equal(t, "https://video-upstream.example/v1/videos/video_123", upstream.requests[1].URL.String())
	require.Equal(t, "https://video-upstream.example/v1/videos/video_123/content", upstream.requests[2].URL.String())
	require.Contains(t, rec.Body.String(), "Video test created: video_123")
	require.Contains(t, rec.Body.String(), `"type":"video"`)
	require.Contains(t, rec.Body.String(), `data:video/mp4;base64,bXA0LWJ5dGVz`)
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestAccountTestService_OpenAIVideoModelUsesReturnedVideoURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(`{"id":"video_456","status":"completed","download_url":"https://cdn.example/video_456.mp4"}`)),
		},
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       58,
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://video-upstream.example/v1",
		},
	}

	err := svc.testOpenAIAccountConnection(c, account, "sora-2", "", "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Contains(t, rec.Body.String(), `"type":"video"`)
	require.Contains(t, rec.Body.String(), `"video_url":"https://cdn.example/video_456.mp4"`)
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestAccountTestService_OpenAIVideoStillProcessingReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalSleep := accountTestSleep
	accountTestSleep = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { accountTestSleep = originalSleep })

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

	upstream := &httpUpstreamRecorder{
		responses: []*http.Response{
			{
				StatusCode: http.StatusAccepted,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(strings.NewReader(`{"id":"video_queued","status":"queued"}`)),
			},
		},
	}
	for i := 0; i < openAIVideoTestPollAttempts; i++ {
		upstream.responses = append(upstream.responses, &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(`{"id":"video_queued","status":"queued"}`)),
		})
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       59,
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://video-upstream.example/v1",
		},
	}

	err := svc.testOpenAIAccountConnection(c, account, "sora-2", "", "")
	require.Error(t, err)
	require.Len(t, upstream.requests, openAIVideoTestPollAttempts+1)
	require.Contains(t, rec.Body.String(), "Video status: queued")
	require.Contains(t, rec.Body.String(), "Video test still processing: video_queued")
	require.NotContains(t, rec.Body.String(), "\"success\":true")
}

func TestAccountTestService_OpenAIMediaOAuthTestsReturnExplicitError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		model     string
		wantError string
	}{
		{
			name:      "audio",
			model:     "gpt-4o-audio-preview",
			wantError: "OpenAI audio test currently supports API Key accounts only",
		},
		{
			name:      "video",
			model:     "sora-2",
			wantError: "OpenAI video test currently supports API Key accounts only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)

			upstream := &httpUpstreamRecorder{}
			svc := &AccountTestService{
				httpUpstream: upstream,
				cfg:          &config.Config{},
			}
			account := &Account{
				ID:       57,
				Name:     "openai-oauth",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"access_token": "test-token",
				},
			}

			err := svc.testOpenAIAccountConnection(c, account, tt.model, "", "")
			require.Error(t, err)
			require.Empty(t, upstream.requests)
			require.Contains(t, rec.Body.String(), tt.wantError)
		})
	}
}

func parseMultipartFormValues(t *testing.T, contentType string, body []byte) map[string]string {
	t.Helper()

	mediaType, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)
	require.NotEmpty(t, params["boundary"])

	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	form, err := reader.ReadForm(1024 * 1024)
	require.NoError(t, err)
	defer func() { _ = form.RemoveAll() }()

	values := make(map[string]string, len(form.Value))
	for key, raw := range form.Value {
		if len(raw) > 0 {
			values[key] = raw[0]
		}
	}
	return values
}
