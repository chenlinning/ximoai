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

func newGatewayRoutesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group:   &service.Group{Platform: service.PlatformOpenAI},
			})
			c.Next()
		}),
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
	)

	return router
}

func TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/responses/compact",
		"/responses/compact",
		"/backend-api/codex/responses",
		"/backend-api/codex/responses/compact",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI responses handler", path)
	}
}

func TestIsOpenAIResponsesPassthroughSubpath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		subpath string
		want    bool
	}{
		{name: "empty", subpath: "", want: false},
		{name: "compact", subpath: "/compact", want: false},
		{name: "compact nested", subpath: "/compact/detail", want: false},
		{name: "response id", subpath: "/resp_123", want: true},
		{name: "response cancel", subpath: "/resp_123/cancel", want: true},
		{name: "input items", subpath: "/resp_123/input_items", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "subpath", Value: tt.subpath}}

			require.Equal(t, tt.want, isOpenAIResponsesPassthroughSubpath(c))
		})
	}
}

func TestGatewayRoutesOpenAIResponsesPassthroughSubpathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/v1/responses/resp_123"},
		{method: http.MethodDelete, path: "/v1/responses/resp_123"},
		{method: http.MethodPost, path: "/v1/responses/resp_123/cancel"},
		{method: http.MethodGet, path: "/responses/resp_123"},
		{method: http.MethodDelete, path: "/responses/resp_123"},
		{method: http.MethodPost, path: "/responses/resp_123/cancel"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "method=%s path=%s should hit OpenAI responses passthrough handler", tt.method, tt.path)
	}
}

func TestGatewayRoutesOpenAIImagesPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/images/generations",
		"/v1/images/edits",
		"/images/generations",
		"/images/edits",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-image-2","prompt":"draw a cat"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI images handler", path)
	}
}

func TestGatewayRoutesOpenAIAudioPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/audio/speech",
		"/v1/audio/transcriptions",
		"/v1/audio/translations",
		"/audio/speech",
		"/audio/transcriptions",
		"/audio/translations",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-4o-mini-tts","input":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI audio handler", path)
	}
}

func TestGatewayRoutesOpenAIRealtimePathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/realtime",
		"/realtime",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI realtime handler", path)
	}
}
