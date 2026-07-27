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

func TestGatewayModels_XimoAIMediaPlatformsNeverFallBackToClaude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []service.Platform{
		{Slug: service.PlatformGrokVideo, Kind: service.PlatformKindGrokVideo, Protocol: service.PlatformProtocolOpenAICompatible, Enabled: true, Builtin: true},
		{Slug: service.PlatformOpenAIAudio, Kind: service.PlatformKindOpenAIAudio, Protocol: service.PlatformProtocolOpenAICompatible, Enabled: true, Builtin: true},
		{Slug: service.PlatformKlingAudio, Kind: service.PlatformKindKlingAudio, Protocol: service.PlatformProtocolOpenAICompatible, Enabled: true, Builtin: true},
	}

	for i, platform := range tests {
		t.Run(platform.Slug, func(t *testing.T) {
			groupID := int64(120 + i)
			h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
				byGroup: map[int64][]service.Account{
					groupID: {{ID: 1, Platform: platform.Slug}},
				},
			})
			h.platformService = service.NewPlatformService(&ximoAIModelsPlatformRepoStub{platform: platform})

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
				Group: &service.Group{ID: groupID, Platform: platform.Slug},
			})

			h.Models(c)

			require.Equal(t, http.StatusOK, rec.Code)
			var got gatewayModelsResponseForTest
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Empty(t, modelIDsForTest(got.Data))
		})
	}
}
