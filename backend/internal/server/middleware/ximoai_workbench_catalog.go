package middleware

import "net/http"

func isWorkbenchCatalogRequest(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	switch path {
	case "/v1/workbench/catalog/groups/available",
		"/v1/workbench/catalog/platforms",
		"/v1/workbench/catalog/model-plaza":
		return true
	default:
		return false
	}
}
