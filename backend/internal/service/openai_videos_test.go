package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIVideoIDFromEndpointSkipsNestedCollections(t *testing.T) {
	require.Equal(t, "video_123", extractOpenAIVideoIDFromEndpoint("/v1/videos/video_123"))
	require.Equal(t, "video_123", extractOpenAIVideoIDFromEndpoint("/v1/videos/video_123/content"))
	require.Equal(t, "video_123", extractOpenAIVideoIDFromEndpoint("/v1/videos/video_123/remix"))
	require.Empty(t, extractOpenAIVideoIDFromEndpoint("/v1/videos/characters/char_123"))
	require.Empty(t, extractOpenAIVideoIDFromEndpoint("/v1/videos/edits"))
	require.Empty(t, extractOpenAIVideoIDFromEndpoint("/v1/videos/extensions"))
}

func TestExtractOpenAIVideoCharacterIDFromEndpoint(t *testing.T) {
	require.Equal(t, "char_123", extractOpenAIVideoCharacterIDFromEndpoint("/v1/videos/characters/char_123"))
	require.Empty(t, extractOpenAIVideoCharacterIDFromEndpoint("/v1/videos/video_123"))
}

func TestParseOpenAIVideoRequestExtractsSourceVideoIDs(t *testing.T) {
	svc := &OpenAIGatewayService{}
	req, err := svc.ParseOpenAIVideoRequest(newGinContextForOpenAIVideoTest("/v1/videos/extensions", "application/json"), []byte(`{"video":{"id":"video_123"},"prompt":"extend"}`), false)
	require.NoError(t, err)
	require.Equal(t, []string{"video_123"}, req.SourceVideoIDs)
}

func TestBuildOpenAIVideosURLHandlesVideosBase(t *testing.T) {
	require.Equal(t, "https://api.example.com/v1/videos/video_123/content", buildOpenAIVideosURL("https://api.example.com/v1/videos", "/v1/videos/video_123/content"))
	require.Equal(t, "https://api.example.com/v1/videos/extensions", buildOpenAIVideosURL("https://api.example.com/v1", "/v1/videos/extensions"))
}

func newGinContextForOpenAIVideoTest(path string, contentType string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("Content-Type", contentType)
	c.Request = req
	return c
}
