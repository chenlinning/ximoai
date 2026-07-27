package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type volcengineAgentPlanRateLimitRepoStub struct {
	AccountRepository
	tempAccountID int64
	tempUntil     time.Time
	tempReason    string
}

func (s *volcengineAgentPlanRateLimitRepoStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	s.tempAccountID = id
	s.tempUntil = until
	s.tempReason = reason
	return nil
}

type volcengineAgentPlanHTTPUpstreamStub struct {
	request *http.Request
	body    []byte
	status  int
	resp    []byte
	reader  io.Reader
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
	responseBody := io.Reader(bytes.NewReader(s.resp))
	if s.reader != nil {
		responseBody = s.reader
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(responseBody),
	}, nil
}

func TestVolcengineAgentPlanUpstreamURLs(t *testing.T) {
	tests := map[VolcengineAgentPlanEndpoint]string{
		VolcengineAgentPlanImagesGenerations:       "https://ark.cn-beijing.volces.com/api/plan/v3/images/generations",
		VolcengineAgentPlanTTSUnidirectional:       "https://openspeech.bytedance.com/api/v3/plan/tts/unidirectional",
		VolcengineAgentPlanTTSUnidirectionalStream: "wss://openspeech.bytedance.com/api/v3/plan/tts/unidirectional/stream",
		VolcengineAgentPlanTTSBidirection:          "wss://openspeech.bytedance.com/api/v3/plan/tts/bidirection",
		VolcengineAgentPlanASRBigmodel:             "wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel",
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

func TestHandleVolcengineAgentPlanUpstreamErrorCoolsDownServerFailures(t *testing.T) {
	repo := &volcengineAgentPlanRateLimitRepoStub{}
	rateLimit := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc := &GatewayService{accountRepo: repo, rateLimitService: rateLimit}
	account := &Account{ID: 77, Platform: PlatformVolcengineAgentPlan, Type: AccountTypeAPIKey}
	err := &VolcengineAgentPlanUpstreamError{
		StatusCode: http.StatusBadGateway,
		Body:       []byte(`{"message":"temporary upstream failure"}`),
		Headers:    http.Header{},
	}

	svc.HandleVolcengineAgentPlanUpstreamError(context.Background(), account, err, VolcengineAgentPlanTTSModel)

	require.Equal(t, int64(77), repo.tempAccountID)
	require.True(t, repo.tempUntil.After(time.Now()))
	require.Contains(t, repo.tempReason, "Volcengine Agent Plan")
}

func TestValidateVolcengineAgentPlanPricingRequiresPositivePerRequestChannelPrice(t *testing.T) {
	groupID := int64(88)
	price := 0.02
	channelService := NewChannelService(&ximoAIModelsChannelRepoStub{channel: Channel{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{groupID},
		ModelPricing: []ChannelModelPricing{{
			Platform:        PlatformVolcengineAgentPlan,
			Models:          []string{VolcengineAgentPlanTTSModel},
			BillingMode:     BillingModePerRequest,
			PerRequestPrice: &price,
		}},
	}, groupPlatforms: map[int64]string{groupID: PlatformVolcengineAgentPlan}}, nil, nil, nil)
	billingService := NewBillingService(&config.Config{}, nil)
	svc := &GatewayService{
		channelService: channelService,
		billingService: billingService,
		resolver:       NewModelPricingResolver(channelService, billingService),
	}
	apiKey := &APIKey{GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformVolcengineAgentPlan}}
	usageFields := ChannelUsageFields{
		OriginalModel:      "doubao-seed-tts-2.0",
		ChannelMappedModel: VolcengineAgentPlanTTSModel,
		BillingModelSource: BillingModelSourceChannelMapped,
	}

	require.NoError(t, svc.ValidateVolcengineAgentPlanPricing(
		context.Background(), apiKey, VolcengineAgentPlanTTSUnidirectional, usageFields, VolcengineAgentPlanTTSModel,
	))

	usageFields.ChannelMappedModel = "missing-price"
	err := svc.ValidateVolcengineAgentPlanPricing(
		context.Background(), apiKey, VolcengineAgentPlanTTSUnidirectional, usageFields, "missing-upstream-price",
	)
	require.ErrorIs(t, err, ErrBillablePricingRequired)
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
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", bytes.NewReader(body))
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

func TestForwardVolcengineAgentPlanTTSPreservesOfficialHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{resp: []byte(`{"code":0,"data":"base64"}{"code":20000000,"usage":{"text_words":5}}`)}
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
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/plan/tts/unidirectional", bytes.NewReader([]byte(`{"text":"hello"}`)))
	c.Request.Header.Set("Authorization", "Bearer client-key-must-not-leak")
	c.Request.Header.Set("X-Api-Resource-Id", "public-seed-tts")
	c.Request.Header.Set("X-Api-Request-Id", "client-request-id")
	c.Request.Header.Set("X-Control-Require-Usage-Tokens-Return", "*")
	c.Request.Header.Set("X-Volcengine-Client-Flag", "keep-me")
	c.Request.Header.Set("Cookie", "main_session=must-not-leak")
	c.Request.Header.Set("Forwarded", "for=203.0.113.1;proto=https")
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.1")
	c.Request.Header.Set("X-Forwarded-Host", "internal.example")
	c.Request.Header.Set("X-Real-Ip", "203.0.113.1")
	c.Request.Header.Set("Cf-Connecting-Ip", "203.0.113.1")

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanTTSUnidirectional, []byte(`{"text":"hello"}`), VolcengineAgentPlanTTSModel,
	)

	require.NoError(t, err)
	require.Equal(t, "tts-key", upstream.request.Header.Get("X-Api-Key"))
	require.Equal(t, VolcengineAgentPlanTTSResourceID, upstream.request.Header.Get("X-Api-Resource-Id"))
	require.Equal(t, "client-request-id", upstream.request.Header.Get("X-Api-Request-Id"))
	require.Empty(t, upstream.request.Header.Get("X-Api-Connect-Id"))
	require.Equal(t, "*", upstream.request.Header.Get("X-Control-Require-Usage-Tokens-Return"))
	require.Equal(t, "keep-me", upstream.request.Header.Get("X-Volcengine-Client-Flag"))
	require.Empty(t, upstream.request.Header.Get("Authorization"))
	require.Empty(t, upstream.request.Header.Get("Cookie"))
	require.Empty(t, upstream.request.Header.Get("Forwarded"))
	require.Empty(t, upstream.request.Header.Get("X-Forwarded-For"))
	require.Empty(t, upstream.request.Header.Get("X-Forwarded-Host"))
	require.Empty(t, upstream.request.Header.Get("X-Real-Ip"))
	require.Empty(t, upstream.request.Header.Get("Cf-Connecting-Ip"))
	require.True(t, recorder.Flushed)
	require.Equal(t, string(upstream.resp), recorder.Body.String())
	require.Zero(t, result.ImageCount)
	require.Zero(t, result.Usage.InputTokens)
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
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, body, VolcengineAgentPlanSeedreamModel,
	)

	require.NoError(t, err)
	require.JSONEq(t, `{"model":"doubao-seedream-5.0-lite","prompt":"mountains","watermark":false}`, string(upstream.body))
	require.Equal(t, VolcengineAgentPlanSeedreamModel, result.UpstreamModel)
}

func TestForwardVolcengineAgentPlanSeedreamStreamsWithoutBuffering(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp: []byte("event: image_generation.partial_succeeded\n" +
			"data: {\"type\":\"image_generation.partial_succeeded\",\"image_index\":0,\"url\":\"https://example.com/1.png\"}\n\n" +
			"event: image_generation.partial_succeeded\n" +
			"data: {\"type\":\"image_generation.partial_succeeded\",\"image_index\":1,\"url\":\"https://example.com/2.png\"}\n\n" +
			"event: image_generation.completed\n" +
			"data: {\"type\":\"image_generation.completed\",\"usage\":{\"generated_images\":2}}\n\n" +
			"data: [DONE]\n\n"),
		headers: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"X-Tt-Logid":   []string{"tt-logid-stream"},
		},
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          13,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "server-side-key"},
	}
	body := []byte(`{"model":"doubao-seedream-5.0-lite","prompt":"mountains","stream":true}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", bytes.NewReader(body))

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, body, VolcengineAgentPlanSeedreamModel,
	)

	require.NoError(t, err)
	require.True(t, result.Stream)
	require.Equal(t, 2, result.ImageCount)
	require.True(t, recorder.Flushed)
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Equal(t, "tt-logid-stream", recorder.Header().Get("X-Tt-Logid"))
	require.Equal(t, "no", recorder.Header().Get("X-Accel-Buffering"))
	require.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))
	require.Equal(t, string(upstream.resp), recorder.Body.String())
}

func TestForwardVolcengineAgentPlanImageResponseCountsReturnedImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp: []byte(`{"data":[{"url":"https://example.com/1.png"},{"url":"https://example.com/2.png"}]}`),
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          15,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "server-side-key"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", bytes.NewReader([]byte(`{"model":"doubao-seedream-5.0-lite","prompt":"mountains"}`)))

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations,
		[]byte(`{"model":"doubao-seedream-5.0-lite","prompt":"mountains"}`), VolcengineAgentPlanSeedreamModel,
	)

	require.NoError(t, err)
	require.Equal(t, 2, result.ImageCount)
}

func TestForwardVolcengineAgentPlanTTSPreservesConcatenatedJSONWithoutTokenCoercion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"code":0,"data":"base64"}{"code":20000000,"usage":{"text_words":123}}`)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{resp: body}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          16,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "tts-key"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/plan/tts/unidirectional", bytes.NewReader(body))

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanTTSUnidirectional, body, VolcengineAgentPlanTTSModel,
	)

	require.NoError(t, err)
	require.Equal(t, body, recorder.Body.Bytes())
	require.Zero(t, result.Usage.InputTokens)
	require.Equal(t, 1, result.RequestCount)
}

func TestForwardVolcengineAgentPlanSeedreamZeroSuccessIsNotBillable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp: []byte("event: image_generation.partial_failed\n" +
			"data: {\"type\":\"image_generation.partial_failed\",\"image_index\":0,\"error\":{\"code\":\"blocked\"}}\n\n" +
			"event: image_generation.completed\n" +
			"data: {\"type\":\"image_generation.completed\",\"usage\":{\"generated_images\":0}}\n\n" +
			"data: [DONE]\n\n"),
		headers: http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 17, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	body := []byte(`{"model":"doubao-seedream-5.0-lite","prompt":"blocked","stream":true}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", bytes.NewReader(body))

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, body, VolcengineAgentPlanSeedreamModel,
	)

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, c.Writer.Written())
	require.Equal(t, string(upstream.resp), recorder.Body.String())
}

func TestForwardVolcengineAgentPlanTTSBusinessFailureIsNotBillable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp: []byte(`{"code":55000000,"message":"resource mismatch"}`),
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 18, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	body := []byte(`{"req_params":{"text":"hello"}}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/plan/tts/unidirectional", bytes.NewReader(body))

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanTTSUnidirectional, body, VolcengineAgentPlanTTSModel,
	)

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, c.Writer.Written())
	require.Equal(t, string(upstream.resp), recorder.Body.String())
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
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", nil)

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, []byte(`{"model":"m"}`), "m",
	)

	require.Nil(t, result)
	var upstreamErr *VolcengineAgentPlanUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.Equal(t, http.StatusTooManyRequests, upstreamErr.StatusCode)
	require.True(t, upstreamErr.Retryable())
	require.False(t, c.Writer.Written())
	events := volcengineAgentPlanOpsEvents(t, c)
	require.Len(t, events, 1)
	require.Equal(t, "http_error", events[0].Kind)
	require.Equal(t, http.StatusTooManyRequests, events[0].UpstreamStatusCode)
	require.Equal(t, int64(9), events[0].AccountID)
}

func TestForwardVolcengineAgentPlanHTTPPreservesProviderErrorInSuccessStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp:    []byte(`{"message":"invalid request"}`),
		headers: http.Header{"X-Api-Status-Code": []string{"45000000"}},
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 12, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v3/plan/tts/unidirectional", nil)

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanTTSUnidirectional, []byte(`{"text":"hello"}`), VolcengineAgentPlanTTSModel,
	)

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, c.Writer.Written())
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "45000000", recorder.Header().Get("X-Api-Status-Code"))
	require.Equal(t, string(upstream.resp), recorder.Body.String())
	events := volcengineAgentPlanOpsEvents(t, c)
	require.Len(t, events, 1)
	require.Equal(t, "business_error", events[0].Kind)
	require.Equal(t, http.StatusOK, events[0].UpstreamStatusCode)
	require.Equal(t, "45000000", events[0].Reason)
}

func TestVolcengineAgentPlanWSHeadersOnlySetUpstreamCredential(t *testing.T) {
	tts := VolcengineAgentPlanWebSocketHeaders(VolcengineAgentPlanTTSBidirection, "tts-key")
	require.Equal(t, "tts-key", tts.Get("X-Api-Key"))
	require.Empty(t, tts.Get("X-Api-Resource-Id"))
	require.Empty(t, tts.Get("X-Api-Connect-Id"))
	require.Empty(t, tts.Get("X-Control-Require-Usage-Tokens-Return"))

	asr := VolcengineAgentPlanWebSocketHeaders(VolcengineAgentPlanASRBigmodelAsync, "asr-key")
	require.Equal(t, "asr-key", asr.Get("X-Api-Key"))
	require.Empty(t, asr.Get("X-Api-Resource-Id"))
	require.Empty(t, asr.Get("X-Api-Request-Id"))
	require.Empty(t, asr.Get("X-Api-Connect-Id"))
	require.Empty(t, asr.Get("X-Api-Sequence"))
}

func TestVolcengineAgentPlanWebSocketResponseHeadersAllowOfficialDiagnosticsOnly(t *testing.T) {
	got := volcengineAgentPlanWebSocketResponseHeaders(http.Header{
		"X-Tt-Logid":       []string{"tt-logid"},
		"X-Api-Connect-Id": []string{"connect-id"},
		"X-Request-Id":     []string{"request-id"},
		"X-Api-Key":        []string{"must-not-leak"},
	})

	require.Equal(t, "tt-logid", got.Get("X-Tt-Logid"))
	require.Equal(t, "connect-id", got.Get("X-Api-Connect-Id"))
	require.Equal(t, "request-id", got.Get("X-Request-Id"))
	require.Empty(t, got.Get("X-Api-Key"))
}

func TestVolcengineAgentPlanResponseObserverParsesSSEStats(t *testing.T) {
	observer := newVolcengineAgentPlanResponseObserver(VolcengineAgentPlanImagesGenerations)
	_, err := observer.Write([]byte("data: {\"type\":\"image_generation.partial_succeeded\",\"image_index\":0,\"url\":\"1.png\"}\n"))
	require.NoError(t, err)
	_, err = observer.Write([]byte("data: {\"type\":\"image_generation.partial_succeeded\",\"image_index\":1,\"url\":\"2.png\"}\n"))
	require.NoError(t, err)
	_, err = observer.Write([]byte("data: {\"type\":\"image_generation.completed\",\"usage\":{\"generated_images\":2}}\n\n"))
	require.NoError(t, err)

	stats := observer.Stats()
	require.Equal(t, 2, stats.ImageCount)
	require.True(t, stats.Completed)
}

func TestVolcengineAgentPlanResponseObserverParsesConcatenatedTTSJSON(t *testing.T) {
	observer := newVolcengineAgentPlanResponseObserver(VolcengineAgentPlanTTSUnidirectional)
	_, err := observer.Write([]byte(`{"code":0,"data":"chunk-1"}{"code":0,"data":"chunk-2"}`))
	require.NoError(t, err)
	_, err = observer.Write([]byte(`{"code":20000000,"usage":{"text_words":45}}`))
	require.NoError(t, err)

	stats := observer.Stats()
	require.True(t, stats.Completed)
	require.False(t, stats.Failed)
}

func TestVolcengineAgentPlanWSUsageTrackerCountsOnlyCompletedSessions(t *testing.T) {
	tracker := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanTTSBidirection)
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(50))
	require.Zero(t, tracker.RequestCount())

	tracker.ObserveClientMessage(volcengineAgentPlanEventFrame(100))
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(150))
	require.Zero(t, tracker.RequestCount())
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(352))
	require.Zero(t, tracker.RequestCount())
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(152))

	tracker.ObserveClientMessage(volcengineAgentPlanEventFrame(100))
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(352))
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(152))

	require.Equal(t, 2, tracker.RequestCount())
	require.False(t, tracker.Failed())
}

func TestVolcengineAgentPlanWSUsageTrackerRejectsErrorAndIncompleteFrames(t *testing.T) {
	errorTracker := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanASRBigmodelAsync)
	errorTracker.ObserveUpstreamMessage(volcengineAgentPlanErrorFrame(45000000, `{"message":"invalid audio"}`))
	require.True(t, errorTracker.Failed())
	require.Zero(t, errorTracker.RequestCount())
	require.Contains(t, errorTracker.FailureMessage(), "45000000")

	partial := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanASRBigmodelNostream)
	partial.ObserveUpstreamMessage(volcengineAgentPlanASRFrame(1, `{"code":0,"result":{"text":"hel"}}`))
	require.False(t, partial.Failed())
	require.Zero(t, partial.RequestCount())

	success := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanASRBigmodelNostream)
	success.ObserveUpstreamMessage(volcengineAgentPlanASRFrame(-1, `{"code":20000000,"result":{"text":"hello"}}`))
	require.Equal(t, 1, success.RequestCount())
	require.False(t, success.Failed())
}

func TestVolcengineAgentPlanWSUsageTrackerWaitsForUnidirectionalTTSFinalEvent(t *testing.T) {
	tracker := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanTTSUnidirectionalStream)
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(350))
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(352))
	require.Zero(t, tracker.RequestCount())

	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(152))
	require.Equal(t, 1, tracker.RequestCount())
}

func TestVolcengineAgentPlanWSUsageTrackerFailureOverridesEarlierCompletion(t *testing.T) {
	tracker := newVolcengineAgentPlanWSUsageTracker(VolcengineAgentPlanTTSBidirection)
	tracker.ObserveClientMessage(volcengineAgentPlanEventFrame(100))
	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(152))
	require.Equal(t, 1, tracker.RequestCount())

	tracker.ObserveUpstreamMessage(volcengineAgentPlanEventFrame(153))
	require.True(t, tracker.Failed())
	require.Zero(t, tracker.RequestCount())
}

func TestForwardVolcengineAgentPlanLargeImageResponseIsNotCapped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const paddingSize = int64(64<<20) + 1024
	prefix := `{"data":[{"b64_json":"`
	suffix := `"}]}`
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		reader: io.MultiReader(strings.NewReader(prefix), &volcengineAgentPlanRepeatReader{remaining: paddingSize, value: 'a'}, strings.NewReader(suffix)),
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 19, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	countingWriter := &volcengineAgentPlanCountingWriter{ResponseWriter: c.Writer}
	c.Writer = countingWriter
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", nil)
	body := []byte(`{"model":"doubao-seedream-5.0-lite","prompt":"mountains"}`)

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, body, VolcengineAgentPlanSeedreamModel,
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, int64(len(prefix)+len(suffix))+paddingSize, countingWriter.written)
}

func TestForwardVolcengineAgentPlanLargeErrorResponseStreamsCompletely(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const payloadSize = int64(3 << 20)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		status:  http.StatusBadGateway,
		reader:  &volcengineAgentPlanRepeatReader{remaining: payloadSize, value: 'x'},
		headers: http.Header{"Retry-After": []string{"9"}},
	}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 20, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	countingWriter := &volcengineAgentPlanCountingWriter{ResponseWriter: c.Writer}
	c.Writer = countingWriter
	c.Request = httptest.NewRequest(http.MethodPost, "/api/plan/v3/images/generations", nil)

	result, err := svc.ForwardVolcengineAgentPlanHTTP(
		context.Background(), c, account, VolcengineAgentPlanImagesGenerations, []byte(`{"model":"m"}`), "m",
	)

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, c.Writer.Written())
	require.Equal(t, http.StatusBadGateway, c.Writer.Status())
	require.Equal(t, "9", c.Writer.Header().Get("Retry-After"))
	require.Equal(t, payloadSize, countingWriter.written)
	events := volcengineAgentPlanOpsEvents(t, c)
	require.Len(t, events, 1)
	require.Equal(t, "http_error", events[0].Kind)
}

func volcengineAgentPlanOpsEvents(t *testing.T, c *gin.Context) []*OpsUpstreamErrorEvent {
	t.Helper()
	raw, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := raw.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	return events
}

func volcengineAgentPlanEventFrame(event int32) []byte {
	frame := []byte{0x11, 0x94, 0x10, 0x00}
	return binary.BigEndian.AppendUint32(frame, uint32(event))
}

func volcengineAgentPlanASRFrame(sequence int32, body string) []byte {
	flags := byte(0x1)
	if sequence < 0 {
		flags = 0x3
	}
	frame := []byte{0x11, 0x90 | flags, 0x10, 0x00}
	frame = binary.BigEndian.AppendUint32(frame, uint32(sequence))
	frame = binary.BigEndian.AppendUint32(frame, uint32(len(body)))
	return append(frame, body...)
}

func volcengineAgentPlanErrorFrame(code uint32, body string) []byte {
	frame := []byte{0x11, 0xf0, 0x10, 0x00}
	frame = binary.BigEndian.AppendUint32(frame, code)
	frame = binary.BigEndian.AppendUint32(frame, uint32(len(body)))
	return append(frame, body...)
}

type volcengineAgentPlanRepeatReader struct {
	remaining int64
	value     byte
}

func (r *volcengineAgentPlanRepeatReader) Read(payload []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(payload)) > r.remaining {
		payload = payload[:r.remaining]
	}
	for i := range payload {
		payload[i] = r.value
	}
	r.remaining -= int64(len(payload))
	return len(payload), nil
}

type volcengineAgentPlanCountingWriter struct {
	gin.ResponseWriter
	written int64
}

func (w *volcengineAgentPlanCountingWriter) Write(payload []byte) (int, error) {
	w.WriteHeaderNow()
	w.written += int64(len(payload))
	return len(payload), nil
}

func (w *volcengineAgentPlanCountingWriter) WriteString(payload string) (int, error) {
	w.WriteHeaderNow()
	w.written += int64(len(payload))
	return len(payload), nil
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

func TestVolcengineAgentPlanAccountTestRejectsProviderErrorHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &volcengineAgentPlanHTTPUpstreamStub{
		resp:    []byte(`{"message":"invalid key scope"}`),
		headers: http.Header{"X-Api-Status-Code": []string{"45000000"}},
	}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:          14,
		Platform:    PlatformVolcengineAgentPlan,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "native-agent-key"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/14/test", nil)

	err := svc.testVolcengineAgentPlanConnection(c, account, VolcengineAgentPlanSeedreamModel, "mountains")

	require.Error(t, err)
	require.NotContains(t, recorder.Body.String(), `"success":true`)
}
