package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXimoAIPlatformRoutesExposeBuiltinManagementOnly(t *testing.T) {
	source, err := os.ReadFile("ximoai.go")
	require.NoError(t, err)

	routes := string(source)
	require.Contains(t, routes, `platforms.GET("", h.Admin.Platform.List)`)
	require.Contains(t, routes, `platforms.PUT("/:slug", h.Admin.Platform.Update)`)
	require.NotContains(t, routes, `platforms.POST("", h.Admin.Platform.Create)`)
	require.NotContains(t, routes, `platforms.DELETE("/:slug", h.Admin.Platform.Delete)`)
}
