package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayService_ForwardAudioPreservesRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-4o-mini-tts","input":"hello","voice":"alloy"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech?format=mp3", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Accept", "audio/mpeg")
	c.Request.Header.Set("OpenAI-Beta", "realtime=v1")
	c.Request.Header.Set("User-Agent", "audio-test")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"audio/mpeg"},
			"X-Request-Id": []string{"rid_audio"},
		},
		Body: io.NopCloser(strings.NewReader("mp3-bytes")),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          77,
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

	result, err := svc.ForwardAudio(context.Background(), c, account, body, &OpenAIAudioRequest{
		Endpoint:    "/v1/audio/speech",
		ContentType: "application/json",
		Model:       "gpt-4o-mini-tts",
		Body:        body,
	}, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "rid_audio", result.RequestID)
	require.Equal(t, "gpt-4o-mini-tts", result.Model)
	require.Equal(t, "https://api.acme.test/v1/audio/speech?format=mp3", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-custom", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "application/json", upstream.lastReq.Header.Get("Content-Type"))
	require.Equal(t, "audio/mpeg", upstream.lastReq.Header.Get("Accept"))
	require.Equal(t, "realtime=v1", upstream.lastReq.Header.Get("OpenAI-Beta"))
	require.Equal(t, "custom-agent", upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, body, upstream.lastBody)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "mp3-bytes", rec.Body.String())
}

func TestOpenAIGatewayService_ForwardAudioMapsKlingInvalidVoiceToBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"kling-audio","input":"hello","voice_id":"bad_voice"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(`{"code":400,"message":"Voice id not found"}`)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:       78,
		Name:     "kling",
		Platform: PlatformKlingAudio,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-kling",
			"base_url": "https://api.kling.test",
		},
	}

	result, err := svc.ForwardAudio(context.Background(), c, account, body, &OpenAIAudioRequest{
		Endpoint:    "/v1/audio/speech",
		ContentType: "application/json",
		Model:       "kling-audio",
		Body:        body,
	}, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid_request_error")
	require.Contains(t, rec.Body.String(), "Voice id not found")
}

func TestOpenAIGatewayService_ParseOpenAIAudioRequestMultipartModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("model", "whisper-1"))
	part, err := writer.CreateFormFile("file", "sample.wav")
	require.NoError(t, err)
	_, err = part.Write([]byte("wav"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/audio/transcriptions", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	parsed, err := (&OpenAIGatewayService{}).ParseOpenAIAudioRequest(c, body.Bytes())

	require.NoError(t, err)
	require.Equal(t, "/v1/audio/transcriptions", parsed.Endpoint)
	require.Equal(t, "whisper-1", parsed.Model)
	require.True(t, parsed.Multipart)
}
