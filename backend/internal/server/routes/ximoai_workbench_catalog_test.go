package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterXimoAIWorkbenchCatalogRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := router.Group("/v1")
	handlers := &handler.Handlers{
		APIKey:           &handler.APIKeyHandler{},
		AvailableChannel: &handler.AvailableChannelHandler{},
		Platform:         &adminhandler.PlatformHandler{},
	}

	registerXimoAIWorkbenchCatalogRoutes(gateway, handlers)

	routes := make(map[string]string)
	for _, route := range router.Routes() {
		routes[route.Path] = route.Method
	}
	require.Equal(t, "GET", routes["/v1/workbench/catalog/groups/available"])
	require.Equal(t, "GET", routes["/v1/workbench/catalog/platforms"])
	require.Equal(t, "GET", routes["/v1/workbench/catalog/model-plaza"])
}
