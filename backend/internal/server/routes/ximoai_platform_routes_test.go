package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXimoAIPlatformManagementRoutesAreRemoved(t *testing.T) {
	source, err := os.ReadFile("ximoai.go")
	require.NoError(t, err)

	routes := string(source)
	require.NotContains(t, routes, `admin.Group("/platforms")`)
	require.NotContains(t, routes, `authenticated.Group("/platforms")`)
	require.NotContains(t, routes, `h.Admin.Platform`)
}
