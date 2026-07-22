package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterWorkbenchRoutesUsesScopedControlTokenLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{
		WorkbenchSSO: handler.NewWorkbenchSSOHandler(nil),
	}
	RegisterWorkbenchRoutes(v1, handlers, middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }))

	routes := make(map[string]string)
	for _, route := range router.Routes() {
		routes[route.Path] = route.Method
	}
	require.Equal(t, "POST", routes["/api/v1/workbench/sso-ticket/validate"])
	require.Equal(t, "POST", routes["/api/v1/workbench/control-token/refresh"])
	require.Equal(t, "POST", routes["/api/v1/workbench/control-token/revoke"])
	require.NotContains(t, routes, "/api/v1/workbench/model-access")
}
