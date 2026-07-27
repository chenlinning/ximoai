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

func TestParseVolcengineAgentPlanRequestUsesOfficialResourceIDForNativeAudio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/plan/tts/unidirectional", strings.NewReader(`{"req_params":{"text":"hello"}}`))
	c.Request.Header.Set("X-Api-Resource-Id", "doubao-seed-tts-2.0")

	body, model, ok := (&GatewayHandler{}).parseVolcengineAgentPlanRequest(c, service.VolcengineAgentPlanTTSUnidirectional)

	require.True(t, ok)
	require.Equal(t, `{"req_params":{"text":"hello"}}`, string(body))
	require.Equal(t, "doubao-seed-tts-2.0", model)
}

func TestParseVolcengineAgentPlanRequestUsesOfficialResourceIDForWebSocketAudio(t *testing.T) {
	tests := []service.VolcengineAgentPlanEndpoint{
		service.VolcengineAgentPlanTTSUnidirectionalStream,
		service.VolcengineAgentPlanTTSBidirection,
		service.VolcengineAgentPlanASRBigmodel,
		service.VolcengineAgentPlanASRBigmodelAsync,
		service.VolcengineAgentPlanASRBigmodelNostream,
	}

	for _, endpoint := range tests {
		t.Run(string(endpoint), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/native-websocket", nil)
			c.Request.Header.Set("X-Api-Resource-Id", "doubao-user-visible-model")

			body, model, ok := (&GatewayHandler{}).parseVolcengineAgentPlanRequest(c, endpoint)

			require.True(t, ok)
			require.Nil(t, body)
			require.Equal(t, "doubao-user-visible-model", model)
		})
	}
}

func TestVolcengineAgentPlanPlatformRecognizesLegacyNativeDefinition(t *testing.T) {
	legacyNative := &GatewayHandler{platformService: service.NewPlatformService(&ximoAIModelsPlatformRepoStub{
		platform: service.Platform{
			Slug:     "volcengine",
			Protocol: service.PlatformProtocolNative,
			BaseURL:  service.PlatformDefaultBaseURLVolcengineAgentPlan,
			Enabled:  true,
			Builtin:  true,
		},
	})}
	require.True(t, legacyNative.IsVolcengineAgentPlanPlatform(context.Background(), "volcengine"))
}

func TestWriteVolcengineAgentPlanErrorPreservesSafeUpstreamHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/plan/tts/unidirectional", nil)
	h := &GatewayHandler{gatewayService: &service.GatewayService{}}

	h.writeVolcengineAgentPlanError(c, &service.VolcengineAgentPlanUpstreamError{
		StatusCode: http.StatusTooManyRequests,
		Body:       []byte(`{"message":"rate limited"}`),
		Headers: http.Header{
			"Content-Type":     []string{"application/json"},
			"Retry-After":      []string{"17"},
			"X-Api-Connect-Id": []string{"connect-id"},
			"X-Tt-Logid":       []string{"log-id"},
			"Set-Cookie":       []string{"upstream=must-not-leak"},
			"X-Api-Key":        []string{"must-not-leak"},
		},
	})

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "17", recorder.Header().Get("Retry-After"))
	require.Equal(t, "connect-id", recorder.Header().Get("X-Api-Connect-Id"))
	require.Equal(t, "log-id", recorder.Header().Get("X-Tt-Logid"))
	require.Empty(t, recorder.Header().Get("Set-Cookie"))
	require.Empty(t, recorder.Header().Get("X-Api-Key"))
}
