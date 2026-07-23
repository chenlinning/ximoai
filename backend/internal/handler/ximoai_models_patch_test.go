package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ximoAIModelsPlatformRepoStub struct {
	service.PlatformRepository
	platform service.Platform
}

func (s *ximoAIModelsPlatformRepoStub) GetBySlug(_ context.Context, slug string) (*service.Platform, error) {
	if service.NormalizePlatformSlug(slug) != s.platform.Slug {
		return nil, service.ErrPlatformNotFound
	}
	platform := s.platform
	return &platform, nil
}

func TestGatewayModels_XimoAICustomPlatformFallbackFollowsProtocol(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		slug      string
		protocol  string
		wantModel string
		notModel  string
	}{
		{
			name:      "openai compatible",
			slug:      "acme-openai",
			protocol:  service.PlatformProtocolOpenAICompatible,
			wantModel: "gpt-5.4",
			notModel:  "claude-sonnet-4-6",
		},
		{
			name:      "gemini",
			slug:      "acme-gemini",
			protocol:  service.PlatformProtocolGemini,
			wantModel: "gemini-2.5-flash",
			notModel:  "claude-sonnet-4-6",
		},
		{
			name:      "anthropic",
			slug:      "acme-anthropic",
			protocol:  service.PlatformProtocolAnthropic,
			wantModel: "claude-sonnet-4-6",
			notModel:  "gemini-2.5-flash",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupID := int64(80 + i)
			h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
				byGroup: map[int64][]service.Account{
					groupID: {{ID: 1, Platform: tt.slug}},
				},
			})
			h.platformService = service.NewPlatformService(&ximoAIModelsPlatformRepoStub{
				platform: service.Platform{
					Slug:     tt.slug,
					Protocol: tt.protocol,
					Enabled:  true,
					Builtin:  false,
				},
			})

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
				Group: &service.Group{ID: groupID, Platform: tt.slug},
			})

			h.Models(c)

			require.Equal(t, http.StatusOK, rec.Code)
			var got gatewayModelsResponseForTest
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			ids := modelIDsForTest(got.Data)
			require.Contains(t, ids, tt.wantModel)
			require.NotContains(t, ids, tt.notModel)
		})
	}
}
