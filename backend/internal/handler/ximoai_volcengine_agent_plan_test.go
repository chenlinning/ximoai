package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newVolcengineAgentPlanHandlerContext(platform, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", strings.NewReader(body))
	groupID := int64(1)
	c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
		ID:      1,
		GroupID: &groupID,
		Group:   &service.Group{ID: groupID, Platform: platform},
	})
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
	return c, recorder
}

func TestVolcengineAgentPlanNativeHandlerRejectsOtherPlatforms(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, recorder := newVolcengineAgentPlanHandlerContext(service.PlatformOpenAI, `{"model":"doubao-seedream-5.0-lite"}`)

	(&GatewayHandler{}).VolcengineAgentPlanImages(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "not supported for this platform")
}

func TestVolcengineAgentPlanImagesRequiresNativeModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, recorder := newVolcengineAgentPlanHandlerContext(service.PlatformVolcengineAgentPlan, `{"prompt":"mountains"}`)

	(&GatewayHandler{}).VolcengineAgentPlanImages(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "model is required")
}

func TestVolcengineAgentPlanPlatformRecognizesLegacyNativeDefinitionOnly(t *testing.T) {
	legacyNative := &GatewayHandler{platformService: service.NewPlatformService(&ximoAIModelsPlatformRepoStub{
		platform: service.Platform{
			Slug:     "volcengine",
			Protocol: service.PlatformProtocolNative,
			BaseURL:  service.PlatformDefaultBaseURLVolcengineAgentPlan,
			Enabled:  true,
			Builtin:  true,
		},
	})}
	customOpenAI := &GatewayHandler{platformService: service.NewPlatformService(&ximoAIModelsPlatformRepoStub{
		platform: service.Platform{
			Slug:     "volcengine",
			Protocol: service.PlatformProtocolOpenAICompatible,
			BaseURL:  "https://ark.cn-beijing.volces.com/api/v3",
			Enabled:  true,
		},
	})}

	require.True(t, legacyNative.IsVolcengineAgentPlanPlatform(context.Background(), "volcengine"))
	require.False(t, customOpenAI.IsVolcengineAgentPlanPlatform(context.Background(), "volcengine"))
	require.True(t, customOpenAI.IsOpenAICompatiblePlatform(context.Background(), "volcengine"))
}
