package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestXimoAIMediaPlatformsRejectUnrelatedOpenAIEntrypoints(t *testing.T) {
	tests := []struct {
		name     string
		platform service.Platform
		allowed  []string
		denied   []string
	}{
		{
			name: "grok video",
			platform: service.Platform{Slug: service.PlatformGrokVideo, Kind: service.PlatformKindGrokVideo,
				Protocol: service.PlatformProtocolOpenAICompatible, Enabled: true, Builtin: true},
			allowed: []string{"/v1/videos/generations"},
			denied:  []string{"/v1/responses", "/v1/messages", "/v1/chat/completions"},
		},
		{
			name: "openai audio",
			platform: service.Platform{Slug: service.PlatformOpenAIAudio, Kind: service.PlatformKindOpenAIAudio,
				Protocol: service.PlatformProtocolOpenAICompatible, Enabled: true, Builtin: true},
			allowed: []string{"/v1/audio/speech", "/v1/audio/transcriptions", "/v1/audio/translations", "/v1/chat/completions"},
			denied:  []string{"/v1/responses", "/v1/messages"},
		},
		{
			name: "kling audio",
			platform: service.Platform{Slug: service.PlatformKlingAudio, Kind: service.PlatformKindKlingAudio,
				Protocol: service.PlatformProtocolOpenAICompatible, Enabled: true, Builtin: true},
			allowed: []string{"/v1/audio/speech"},
			denied:  []string{"/v1/audio/transcriptions", "/v1/audio/translations", "/v1/responses", "/v1/messages", "/v1/chat/completions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newGatewayPlatformBoundaryRouter(tt.platform)
			for _, path := range tt.allowed {
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"test-model","input":"hi","messages":[{"role":"user","content":"hi"}]}`))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s body=%s", path, w.Body.String())
			}
			for _, path := range tt.denied {
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"test-model","input":"hi","messages":[{"role":"user","content":"hi"}]}`))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				require.Equal(t, http.StatusNotFound, w.Code, "path=%s body=%s", path, w.Body.String())
			}
		})
	}
}

func TestXimoAIMediaPlatformRoutesSurviveRename(t *testing.T) {
	tests := []struct {
		name     string
		platform service.Platform
		path     string
	}{
		{
			name: "grok video",
			platform: service.Platform{Slug: "renamed-video", Kind: service.PlatformKindGrokVideo,
				Protocol: service.PlatformProtocolGemini, Enabled: true, Builtin: true},
			path: "/v1/videos/generations",
		},
		{
			name: "openai audio",
			platform: service.Platform{Slug: "renamed-openai-audio", Kind: service.PlatformKindOpenAIAudio,
				Protocol: service.PlatformProtocolGemini, Enabled: true, Builtin: true},
			path: "/v1/audio/speech",
		},
		{
			name: "kling audio",
			platform: service.Platform{Slug: "renamed-kling", Kind: service.PlatformKindKlingAudio,
				Protocol: service.PlatformProtocolGemini, Enabled: true, Builtin: true},
			path: "/v1/audio/speech",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newGatewayPlatformBoundaryRouter(tt.platform)
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(`{"model":"test-model","input":"hi"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.NotEqual(t, http.StatusNotFound, w.Code, w.Body.String())
		})
	}
}
