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
	require.Equal(t, "operations/video_123", extractOpenAIVideoIDFromEndpoint("/v1/videos/operations%2Fvideo_123"))
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

func TestBuildGeminiVideoURLHandlesVersionedBase(t *testing.T) {
	require.Equal(t, "https://generativelanguage.googleapis.com/v1beta/models/veo:generateVideos", buildGeminiVideoURL("https://generativelanguage.googleapis.com", "/v1beta/models/veo:generateVideos"))
	require.Equal(t, "https://generativelanguage.googleapis.com/v1beta/operations/video_123", buildGeminiVideoURL("https://generativelanguage.googleapis.com/v1beta", "/v1beta/operations/video_123"))
	require.Equal(t, "https://gemini.example.com/v1beta/operations/video_123", buildGeminiVideoURL("https://gemini.example.com/v1beta/models", "/v1beta/operations/video_123"))
}

func TestExtractOpenAIVideoIDReadsGeminiOperationName(t *testing.T) {
	require.Equal(t, "operations/video_123", extractOpenAIVideoID([]byte(`{"name":"operations/video_123","done":false}`)))
	require.Equal(t, "operations/video_456", extractOpenAIVideoID([]byte(`{"operation":{"name":"operations/video_456"}}`)))
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
