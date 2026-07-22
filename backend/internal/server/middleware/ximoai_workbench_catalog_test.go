package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkbenchCatalogRequestUsesAnExactReadOnlyAllowlist(t *testing.T) {
	for _, path := range []string{
		"/v1/workbench/catalog/groups/available",
		"/v1/workbench/catalog/platforms",
		"/v1/workbench/catalog/model-plaza",
	} {
		require.True(t, isWorkbenchCatalogRequest(http.MethodGet, path))
		require.False(t, isWorkbenchCatalogRequest(http.MethodPost, path))
	}
	require.False(t, isWorkbenchCatalogRequest(http.MethodGet, "/v1/workbench/catalog"))
	require.False(t, isWorkbenchCatalogRequest(http.MethodGet, "/v1/workbench/catalog/platforms/extra"))
}
