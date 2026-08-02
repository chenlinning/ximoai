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

func TestRegisterWorkbenchRoutesUsesScopedControlTokenLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{
		WorkbenchSSO: handler.NewWorkbenchSSOHandler(nil, nil),
	}
	RegisterWorkbenchRoutes(v1, handlers, middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }), nil)

	routes := make(map[string]string)
	for _, route := range router.Routes() {
		routes[route.Path] = route.Method
	}
	require.Equal(t, "POST", routes["/api/v1/workbench/sso-ticket/validate"])
	require.Equal(t, "POST", routes["/api/v1/workbench/control-token/refresh"])
	require.Equal(t, "POST", routes["/api/v1/workbench/control-token/revoke"])
	require.NotContains(t, routes, "/api/v1/workbench/model-access")
}

func TestWorkbenchInternalRoutesRateLimitFailCloseWhenRedisUnavailable(t *testing.T) {
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
	RegisterWorkbenchRoutes(v1, &handler.Handlers{WorkbenchSSO: handler.NewWorkbenchSSOHandler(nil, nil)}, middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }), redisClient)

	for _, path := range []string{
		"/api/v1/workbench/sso-ticket/validate",
		"/api/v1/workbench/control-token/refresh",
		"/api/v1/workbench/control-token/revoke",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusTooManyRequests, w.Code, "path=%s", path)
		require.Contains(t, w.Body.String(), "rate limit exceeded", "path=%s", path)
	}
}
