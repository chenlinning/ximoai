package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRegisterDesktopRoutesExposesOnlyUnifiedDesktopEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{DesktopSession: handler.NewDesktopSessionHandler(nil, nil)}
	RegisterDesktopRoutes(v1, handlers, middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }), nil)

	registered := map[string]string{}
	for _, route := range router.Routes() {
		registered[route.Path] = route.Method
	}
	require.Equal(t, http.MethodPost, registered["/api/v1/desktop/authorize"])
	require.Equal(t, http.MethodPost, registered["/api/v1/desktop/token"])
	require.Equal(t, http.MethodPost, registered["/api/v1/desktop/sso-ticket"])
	require.Equal(t, http.MethodPost, registered["/api/v1/desktop/sso-broker-credential"])
	require.Equal(t, http.MethodDelete, registered["/api/v1/desktop/session"])
	require.Equal(t, http.MethodPost, registered["/api/v1/desktop/revoke"])
	require.NotContains(t, registered, "/api/v1/desktop/audience")
}

func TestDesktopRoutesRateLimitFailCloseWhenRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() { _ = redisClient.Close() })

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterDesktopRoutes(v1, &handler.Handlers{DesktopSession: handler.NewDesktopSessionHandler(nil, nil)}, middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }), redisClient)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/desktop/authorize"},
		{http.MethodPost, "/api/v1/desktop/token"},
		{http.MethodPost, "/api/v1/desktop/sso-ticket"},
		{http.MethodPost, "/api/v1/desktop/sso-broker-credential"},
		{http.MethodDelete, "/api/v1/desktop/session"},
		{http.MethodPost, "/api/v1/desktop/revoke"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusTooManyRequests, w.Code, "path=%s", tc.path)
		require.Contains(t, w.Body.String(), "rate limit exceeded", "path=%s", tc.path)
	}
}
