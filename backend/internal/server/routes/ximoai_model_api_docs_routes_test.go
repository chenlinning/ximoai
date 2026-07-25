package routes

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestXimoAIModelAPIDocsProfilesReferenceRegisteredGatewayRoutes(t *testing.T) {
	router := newGatewayRoutesTestRouter(service.PlatformVolcengineAgentPlan)
	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	for _, profile := range handler.XimoAIModelAPIDocsProfiles() {
		for _, variant := range profile.Variants {
			for _, step := range variant.Steps {
				if step.RoutePath == "" {
					continue
				}
				_, ok := registered[step.Method+" "+step.RoutePath]
				require.True(t, ok, "%s/%s references unregistered route %s %s", profile.ID, variant.ID, step.Method, step.RoutePath)
			}
		}
	}
}

func TestXimoAIModelAPIDocsRelevantGatewayRoutesAreReviewed(t *testing.T) {
	router := newGatewayRoutesTestRouter(service.PlatformVolcengineAgentPlan)
	documented := make(map[string]struct{})
	for _, profile := range handler.XimoAIModelAPIDocsProfiles() {
		for _, variant := range profile.Variants {
			for _, step := range variant.Steps {
				if step.RoutePath != "" {
					documented[step.Method+" "+step.RoutePath] = struct{}{}
				}
			}
		}
	}

	// These routes are intentionally outside the per-model documentation dialog.
	excluded := map[string]string{
		"POST /v1/messages/count_tokens":                      "token utility",
		"POST /v1/responses/*subpath":                         "Responses auxiliary operations",
		"GET /v1beta/models":                                  "model discovery",
		"GET /v1beta/models/:model":                           "model discovery",
		"POST /v1/images/batches":                             "batch job management",
		"GET /v1/images/batches":                              "batch job management",
		"GET /v1/images/batches/models":                       "batch job management",
		"GET /v1/images/batches/:id":                          "batch job management",
		"GET /v1/images/batches/:id/items":                    "batch job management",
		"GET /v1/images/batches/:id/items/:custom_id/content": "batch job management",
		"GET /v1/images/batches/:id/download":                 "batch job management",
		"POST /v1/images/batches/:id/cancel":                  "batch job management",
		"DELETE /v1/images/batches/:id":                       "batch job management",
		"DELETE /v1/images/batches/:id/outputs":               "batch job management",
		"POST /v1/videos":                                     "unsupported compatibility route",
		"GET /v1/videos":                                      "unsupported compatibility route",
	}

	for _, route := range router.Routes() {
		if !modelAPIDocsRelevantRoutePath(route.Path) {
			continue
		}
		key := route.Method + " " + route.Path
		if _, ok := documented[key]; ok {
			continue
		}
		_, ok := excluded[key]
		require.True(t, ok, "%s is a new or changed model route; add a documentation profile or an explicit reviewed exclusion", key)
	}
}

func modelAPIDocsRelevantRoutePath(path string) bool {
	for _, prefix := range []string{
		"/v1/messages",
		"/v1/responses",
		"/v1/chat/completions",
		"/v1/images",
		"/v1/videos",
		"/v1/audio",
		"/v1/volcengine",
		"/v1beta/models",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
