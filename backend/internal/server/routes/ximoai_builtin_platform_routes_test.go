package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGatewayPlatformBoundaryRouter(platform string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.Gateway.MaxBodySize = 1 << 20
	gatewayHandler := handler.NewGatewayHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil,
		nil,
		cfg,
		nil,
	)

	router := gin.New()
	router.Use(gin.Recovery())
	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       gatewayHandler,
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group:   &service.Group{ID: groupID, Platform: platform, AllowMessagesDispatch: true},
			})
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
			c.Next()
		}),
		nil, nil, nil, nil, nil,
		cfg,
	)
	return router
}

func TestXimoAIMediaPlatformsRejectUnrelatedOpenAIEntrypoints(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		allowed  []string
		denied   []string
	}{
		{
			name:     "grok video",
			platform: service.PlatformGrokVideo,
			allowed:  []string{"/v1/videos/generations"},
			denied:   []string{"/v1/responses", "/v1/messages", "/v1/chat/completions"},
		},
		{
			name:     "openai audio",
			platform: service.PlatformOpenAIAudio,
			allowed:  []string{"/v1/audio/speech", "/v1/audio/transcriptions", "/v1/audio/translations", "/v1/chat/completions"},
			denied:   []string{"/v1/responses", "/v1/messages"},
		},
		{
			name:     "kling audio",
			platform: service.PlatformKlingAudio,
			allowed:  []string{"/v1/audio/speech"},
			denied:   []string{"/v1/audio/transcriptions", "/v1/audio/translations", "/v1/responses", "/v1/messages", "/v1/chat/completions"},
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
