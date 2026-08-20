package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayModels_XimoAIMediaPlatformsNeverFallBackToClaude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []string{
		service.PlatformGrokVideo,
		service.PlatformOpenAIAudio,
		service.PlatformKlingAudio,
	}

	for i, platform := range tests {
		t.Run(platform, func(t *testing.T) {
			groupID := int64(120 + i)
			h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
				byGroup: map[int64][]service.Account{
					groupID: {{ID: 1, Platform: platform}},
				},
			})

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
				Group: &service.Group{ID: groupID, Platform: platform},
			})

			h.Models(c)

			require.Equal(t, http.StatusOK, rec.Code)
			var got gatewayModelsResponseForTest
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Empty(t, modelIDsForTest(got.Data))
		})
	}
}
