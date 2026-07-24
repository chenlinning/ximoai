package routes

import (
	"context"
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

type gatewayPlatformBoundaryRepo struct {
	platform service.Platform
}

func (r *gatewayPlatformBoundaryRepo) List(context.Context, bool) ([]service.Platform, error) {
	return []service.Platform{r.platform}, nil
}

func (r *gatewayPlatformBoundaryRepo) GetBySlug(_ context.Context, slug string) (*service.Platform, error) {
	if service.NormalizePlatformSlug(slug) != r.platform.Slug {
		return nil, service.ErrPlatformNotFound
	}
	platform := r.platform
	return &platform, nil
}

func (*gatewayPlatformBoundaryRepo) Create(context.Context, *service.Platform) error { return nil }
func (*gatewayPlatformBoundaryRepo) Update(context.Context, *service.Platform) error { return nil }
func (*gatewayPlatformBoundaryRepo) Rename(context.Context, string, *service.Platform) error {
	return nil
}
func (*gatewayPlatformBoundaryRepo) Delete(context.Context, string) error { return nil }
func (*gatewayPlatformBoundaryRepo) Usage(context.Context, string) (service.PlatformUsage, error) {
	return service.PlatformUsage{}, nil
}

func newGatewayPlatformBoundaryRouter(platform service.Platform) *gin.Engine {
	gin.SetMode(gin.TestMode)
	platformService := service.NewPlatformService(&gatewayPlatformBoundaryRepo{platform: platform})
	cfg := &config.Config{}
	cfg.Gateway.MaxBodySize = 1 << 20
	gatewayHandler := handler.NewGatewayHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil,
		platformService,
		nil,
		cfg,
		nil,
	)

	router := gin.New()
	router.Use(gin.Recovery())
	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       gatewayHandler,
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group:   &service.Group{ID: groupID, Platform: platform.Slug, AllowMessagesDispatch: true},
			})
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
			c.Next()
		}),
		nil, nil, nil, nil, nil,
		cfg,
	)
	return router
}

func TestGatewayRoutesCustomOpenAICompatiblePlatformUsesOpenAIResponsesHandler(t *testing.T) {
	router := newGatewayPlatformBoundaryRouter(service.Platform{
		Slug:      "acme-openai",
		Protocol:  service.PlatformProtocolOpenAICompatible,
		AuthModes: []string{service.AccountTypeAPIKey},
		Enabled:   true,
		Builtin:   false,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"custom-model","input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "Service temporarily unavailable")
}

func TestGatewayRoutesCustomOpenAICompatiblePlatformReusesOpenAIAPIKeyEntrypoints(t *testing.T) {
	router := newGatewayPlatformBoundaryRouter(service.Platform{
		Slug:      "acme-openai",
		Protocol:  service.PlatformProtocolOpenAICompatible,
		AuthModes: []string{service.AccountTypeAPIKey},
		Enabled:   true,
		Builtin:   false,
	})

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "embeddings", path: "/v1/embeddings", body: `{"model":"text-embedding-test","input":"hello"}`},
		{name: "images", path: "/v1/images/generations", body: `{"model":"image-test","prompt":"hello"}`},
		{name: "count tokens", path: "/v1/messages/count_tokens", body: `{"model":"custom-model","messages":[{"role":"user","content":"hello"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), "Service temporarily unavailable")
			require.NotContains(t, w.Body.String(), "not supported for this platform")
		})
	}
}
