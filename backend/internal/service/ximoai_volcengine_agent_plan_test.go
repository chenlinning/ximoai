package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type volcengineAgentPlanHTTPUpstreamStub struct {
	request *http.Request
	body    []byte
	status  int
	resp    []byte
	headers http.Header
}

func (s *volcengineAgentPlanHTTPUpstreamStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return s.capture(req)
}

func (s *volcengineAgentPlanHTTPUpstreamStub) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.capture(req)
}

func (s *volcengineAgentPlanHTTPUpstreamStub) capture(req *http.Request) (*http.Response, error) {
	s.request = req.Clone(req.Context())
	s.body, _ = io.ReadAll(req.Body)
	status := s.status
	if status == 0 {
		status = http.StatusOK
	}
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Request-Id": []string{"volc-request-1"},
	}
	for key, values := range s.headers {
		headers[key] = append([]string(nil), values...)
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(s.resp)),
	}, nil
}

func TestVolcengineAgentPlanUpstreamURLs(t *testing.T) {
	tests := map[VolcengineAgentPlanEndpoint]string{
		VolcengineAgentPlanImagesGenerations:       "https://ark.cn-beijing.volces.com/api/plan/v3/images/generations",
		VolcengineAgentPlanTTSUnidirectional:       "https://openspeech.bytedance.com/api/v3/plan/tts/unidirectional",
		VolcengineAgentPlanTTSUnidirectionalStream: "wss://openspeech.bytedance.com/api/v3/plan/tts/unidirectional/stream",
		VolcengineAgentPlanTTSBidirection:          "wss://openspeech.bytedance.com/api/v3/plan/tts/bidirection",
		VolcengineAgentPlanASRBigmodelAsync:        "wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_async",
		VolcengineAgentPlanASRBigmodelNostream:     "wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_nostream",
	}

	for endpoint, want := range tests {
		t.Run(string(endpoint), func(t *testing.T) {
			got, err := VolcengineAgentPlanUpstreamURL(endpoint)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestForwardVolcengineAgentPlanHTTPPreservesBodyAndReplacesAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{resp: []byte(`{"data":[{"url":"https://example.com/image.png"}]}`)}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          7,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "server-side-key"},
	}
	body := []byte(`{"model":"doubao-seedream-5.0-lite","prompt":"mountains"}`)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/volcengine/images/generations", bytes.NewReader(body))
	c.Request.Header.Set("Authorization", "Bearer client-key-must-not-leak")
	c.Request.Header.Set("X-Api-Key", "client-key-must-not-leak")
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, body, "doubao-seedream-5.0-lite",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, body, upstream.body)
	require.Equal(t, "Bearer server-side-key", upstream.request.Header.Get("Authorization"))
	require.Empty(t, upstream.request.Header.Get("X-Api-Key"))
	require.Equal(t, "application/json", upstream.request.Header.Get("Content-Type"))
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, 1, result.RequestCount)
	require.Equal(t, "doubao-seedream-5.0-lite", result.Model)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestForwardVolcengineAgentPlanTTSUsesPlanHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{resp: []byte(`{"audio":"base64"}`)}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          8,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "tts-key"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/volcengine/audio/tts/unidirectional", bytes.NewReader([]byte(`{"text":"hello"}`)))

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanTTSUnidirectional, []byte(`{"text":"hello"}`), VolcengineAgentPlanTTSModel,
	)

	require.NoError(t, err)
	require.Equal(t, "tts-key", upstream.request.Header.Get("X-Api-Key"))
	require.Equal(t, VolcengineAgentPlanTTSResourceID, upstream.request.Header.Get("X-Api-Resource-Id"))
	require.NotEmpty(t, upstream.request.Header.Get("X-Api-Connect-Id"))
	require.Equal(t, "*", upstream.request.Header.Get("X-Control-Require-Usage-Tokens-Return"))
	require.Empty(t, upstream.request.Header.Get("Authorization"))
	require.Zero(t, result.ImageCount)
	require.Equal(t, 1, result.RequestCount)
}

func TestForwardVolcengineAgentPlanImageMappingOnlyRewritesModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{resp: []byte(`{"data":[{"url":"https://example.com/image.png"}]}`)}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          11,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "server-side-key"},
	}
	body := []byte(`{"model":"public-seedream","prompt":"mountains","watermark":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/volcengine/images/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, body, VolcengineAgentPlanSeedreamModel,
	)

	require.NoError(t, err)
	require.JSONEq(t, `{"model":"doubao-seedream-5.0-lite","prompt":"mountains","watermark":false}`, string(upstream.body))
	require.Equal(t, VolcengineAgentPlanSeedreamModel, result.UpstreamModel)
}

func TestForwardVolcengineAgentPlanHTTPDoesNotCommitUpstreamFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		status: http.StatusTooManyRequests,
		resp:   []byte(`{"error":{"message":"rate limited"}}`),
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 9, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/volcengine/images/generations", nil)

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, []byte(`{"model":"m"}`), "m",
	)

	require.Nil(t, result)
	var upstreamErr *VolcengineAgentPlanUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.Equal(t, http.StatusTooManyRequests, upstreamErr.StatusCode)
	require.True(t, upstreamErr.Retryable())
	require.False(t, c.Writer.Written())
}

func TestForwardVolcengineAgentPlanHTTPRejectsProviderErrorInSuccessStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp:    []byte(`{"message":"invalid request"}`),
		headers: http.Header{"X-Api-Status-Code": []string{"45000000"}},
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 12, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/volcengine/audio/tts/unidirectional", nil)

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanTTSUnidirectional, []byte(`{"text":"hello"}`), VolcengineAgentPlanTTSModel,
	)

	require.Nil(t, result)
	var upstreamErr *VolcengineAgentPlanUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.Equal(t, "45000000", upstreamErr.ProviderCode)
	require.Equal(t, http.StatusBadGateway, upstreamErr.StatusCode)
	require.False(t, c.Writer.Written())
}

func TestVolcengineAgentPlanWSHeaders(t *testing.T) {
	tts := VolcengineAgentPlanWebSocketHeaders(VolcengineAgentPlanTTSBidirection, "tts-key")
	require.Equal(t, "tts-key", tts.Get("X-Api-Key"))
	require.Equal(t, VolcengineAgentPlanTTSResourceID, tts.Get("X-Api-Resource-Id"))
	require.NotEmpty(t, tts.Get("X-Api-Connect-Id"))
	require.Equal(t, "*", tts.Get("X-Control-Require-Usage-Tokens-Return"))

	asr := VolcengineAgentPlanWebSocketHeaders(VolcengineAgentPlanASRBigmodelAsync, "asr-key")
	require.Equal(t, "asr-key", asr.Get("X-Api-Key"))
	require.Equal(t, VolcengineAgentPlanASRResourceID, asr.Get("X-Api-Resource-Id"))
	require.NotEmpty(t, asr.Get("X-Api-Request-Id"))
	require.NotEmpty(t, asr.Get("X-Api-Connect-Id"))
	require.Equal(t, "-1", asr.Get("X-Api-Sequence"))
}

func TestVolcengineAgentPlanWSUsageTrackerCountsLogicalTasksOnce(t *testing.T) {
	tracker := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanTTSBidirection)
	tracker.ObserveUpstreamMessage([]byte(`{"event":"ConnectionStarted"}`))
	require.Zero(t, tracker.RequestCount())

	tracker.ObserveClientMessage([]byte(`{"event":"StartSession"}`))
	tracker.ObserveUpstreamMessage([]byte(`{"event":"SessionStarted"}`))
	require.Zero(t, tracker.RequestCount())
	tracker.ObserveUpstreamMessage([]byte(`{"event":"audio","data":"chunk-1"}`))
	tracker.ObserveUpstreamMessage([]byte(`{"event":"audio","data":"chunk-2"}`))
	tracker.ObserveClientMessage([]byte(`{"event":"StartSession"}`))
	tracker.ObserveUpstreamMessage([]byte(`{"event":"audio","data":"chunk-3"}`))

	require.Equal(t, 2, tracker.RequestCount())

	failed := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanASRBigmodelAsync)
	failed.ObserveUpstreamMessage([]byte(`{"error":{"message":"invalid audio"}}`))
	require.Zero(t, failed.RequestCount())

	success := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanASRBigmodelNostream)
	success.ObserveUpstreamMessage([]byte(`{"code":20000000,"result":{"text":"hello"}}`))
	require.Equal(t, 1, success.RequestCount())
}

func TestVolcengineAgentPlanAccountTestUsesNativeSeedreamEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp: []byte(`{"data":[{"url":"https://example.com/seedream.png"}]}`),
	}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:          10,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "native-agent-key"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/10/test", nil)

	err := svc.testVolcengineAgentPlanConnection(c, account, VolcengineAgentPlanSeedreamModel, "mountains")

	require.NoError(t, err)
	require.Equal(t, "https://ark.cn-beijing.volces.com/api/plan/v3/images/generations", upstream.request.URL.String())
	require.Equal(t, "Bearer native-agent-key", upstream.request.Header.Get("Authorization"))
	require.Contains(t, recorder.Body.String(), "test_complete")
	require.Contains(t, recorder.Body.String(), "seedream.png")
}
