package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptOpenAIVideoProviderRequestMapsGrokCreate(t *testing.T) {
	account := &Account{Platform: PlatformGrokVideo}
	body := []byte(`{
		"model":"grok-video-3-10s",
		"prompt":"city at sunset",
		"aspect_ratio":"16:9",
		"size":"720P",
		"images":["https://example.com/ref.png"]
	}`)

	rewritten, err := adaptOpenAIVideoProviderRequest(account, http.MethodPost, "/v1/videos", body, "application/json")

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, rewritten.Method)
	require.Equal(t, "/v1/video/create", rewritten.Endpoint)
	require.Equal(t, "application/json", rewritten.ContentType)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rewritten.Body, &payload))
	require.Equal(t, "grok-video-3-10s", payload["model"])
	require.Equal(t, "city at sunset", payload["prompt"])
	require.Equal(t, "16:9", payload["aspect_ratio"])
	require.Equal(t, "720P", payload["size"])
	require.Equal(t, []any{"https://example.com/ref.png"}, payload["images"])
}

func TestAdaptOpenAIVideoProviderRequestMapsGrokQuery(t *testing.T) {
	account := &Account{Platform: PlatformGrokVideo}

	rewritten, err := adaptOpenAIVideoProviderRequest(account, http.MethodGet, "/v1/videos/task_123", nil, "")

	require.NoError(t, err)
	require.Equal(t, http.MethodGet, rewritten.Method)
	require.Equal(t, "/v1/video/query?id=task_123", rewritten.Endpoint)
}

func TestAdaptOpenAIVideoProviderRequestMapsGrokExtend(t *testing.T) {
	account := &Account{Platform: PlatformGrokVideo}
	body := []byte(`{"model":"grok-video-3-10s","prompt":"extend it","video":{"id":"task_123"},"start_time":10}`)

	rewritten, err := adaptOpenAIVideoProviderRequest(account, http.MethodPost, "/v1/videos/extensions", body, "application/json")

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, rewritten.Method)
	require.Equal(t, "/v1/video/extend", rewritten.Endpoint)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rewritten.Body, &payload))
	require.Equal(t, "task_123", payload["task_id"])
	require.Equal(t, "extend it", payload["prompt"])
	require.Equal(t, float64(10), payload["start_time"])
}

func TestAdaptOpenAIVideoProviderRequestMapsGeminiGenerateVideos(t *testing.T) {
	account := &Account{Platform: PlatformGemini, Type: AccountTypeAPIKey}
	body := []byte(`{
		"model":"veo-3.1-generate-preview",
		"prompt":"city at sunset",
		"aspect_ratio":"16:9",
		"duration_seconds":8,
		"n":2,
		"negative_prompt":"rain"
	}`)

	rewritten, err := adaptOpenAIVideoProviderRequest(account, http.MethodPost, "/v1/videos", body, "application/json")

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, rewritten.Method)
	require.Equal(t, "/v1beta/models/veo-3.1-generate-preview:generateVideos", rewritten.Endpoint)
	require.Equal(t, "application/json", rewritten.ContentType)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rewritten.Body, &payload))
	require.Equal(t, "city at sunset", payload["prompt"])
	require.Equal(t, "rain", payload["negativePrompt"])
	config, ok := payload["config"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "16:9", config["aspectRatio"])
	require.Equal(t, float64(8), config["durationSeconds"])
	require.Equal(t, float64(2), config["numberOfVideos"])
}

func TestAdaptOpenAIVideoProviderRequestMapsGeminiOperationQuery(t *testing.T) {
	account := &Account{Platform: PlatformGemini, Type: AccountTypeAPIKey}

	rewritten, err := adaptOpenAIVideoProviderRequest(account, http.MethodGet, "/v1/videos/operations%2Fvideo_123", nil, "")

	require.NoError(t, err)
	require.Equal(t, http.MethodGet, rewritten.Method)
	require.Equal(t, "/v1beta/operations/video_123", rewritten.Endpoint)
}

func TestAdaptOpenAIAudioProviderRequestMapsKlingTTS(t *testing.T) {
	account := &Account{Platform: "kling_audio"}
	body := []byte(`{"model":"kling-audio-tts","input":"你好","voice":"voice_1","voice_language":"zh","voice_speed":1.1}`)

	rewritten, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, body, "application/json")

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, rewritten.Method)
	require.Equal(t, "/kling/v1/audio/tts", rewritten.Endpoint)
	require.Equal(t, "application/json", rewritten.ContentType)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rewritten.Body, &payload))
	require.Equal(t, "你好", payload["text"])
	require.Equal(t, "voice_1", payload["voice_id"])
	require.Equal(t, "zh", payload["voice_language"])
	require.Equal(t, 1.1, payload["voice_speed"])
}

func TestAdaptOpenAIAudioProviderRequestMapsKlingPublicAudioAlias(t *testing.T) {
	account := &Account{Platform: "kling_audio"}
	body := []byte(`{"model":"kling-audio","input":"hello","voice":"voice_1","voice_language":"zh"}`)

	rewritten, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, body, "application/json")

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, rewritten.Method)
	require.Equal(t, "/kling/v1/audio/tts", rewritten.Endpoint)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rewritten.Body, &payload))
	require.Equal(t, "hello", payload["text"])
	require.Equal(t, "voice_1", payload["voice_id"])
}

func TestAdaptOpenAIAudioProviderRequestRejectsOpenAIVoiceNamesForKling(t *testing.T) {
	account := &Account{Platform: "kling_audio"}
	body := []byte(`{"model":"kling-audio","input":"hello","voice":"alloy"}`)

	_, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, body, "application/json")

	require.Error(t, err)
	require.Contains(t, err.Error(), "kling voice_id is required")
}

func TestAdaptOpenAIAudioProviderRequestMapsKlingCustomVoiceCreateAndQuery(t *testing.T) {
	account := &Account{Platform: "kling_audio"}
	createBody := []byte(`{"model":"kling-custom-voices","voice_name":"demo","voice_url":"https://example.com/a.wav"}`)

	createReq, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, createBody, "application/json")
	require.NoError(t, err)
	require.Equal(t, "/kling/v1/general/custom-voices", createReq.Endpoint)

	queryBody := []byte(`{"model":"kling-custom-voices","voice_id":"voice_123"}`)
	queryReq, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, queryBody, "application/json")
	require.NoError(t, err)
	require.Equal(t, http.MethodGet, queryReq.Method)
	require.Equal(t, "/kling/v1/general/custom-voices/voice_123", queryReq.Endpoint)
}

func TestAdaptOpenAIAudioProviderRequestMapsKlingPresetVoices(t *testing.T) {
	account := &Account{Platform: "kling_audio"}
	body := []byte(`{"model":"kling-presets-voices","pageNum":2,"pageSize":5}`)

	rewritten, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, body, "application/json")

	require.NoError(t, err)
	require.Equal(t, http.MethodGet, rewritten.Method)
	require.Equal(t, "/kling/v1/general/presets-voices?pageNum=2&pageSize=5", rewritten.Endpoint)
}
