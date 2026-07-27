package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type VolcengineAgentPlanEndpoint string

var ErrBillablePricingRequired = errors.New("billable usage requires positive pricing")

const (
	VolcengineAgentPlanImagesGenerations       VolcengineAgentPlanEndpoint = "images_generations"
	VolcengineAgentPlanTTSUnidirectional       VolcengineAgentPlanEndpoint = "tts_unidirectional"
	VolcengineAgentPlanTTSUnidirectionalStream VolcengineAgentPlanEndpoint = "tts_unidirectional_stream"
	VolcengineAgentPlanTTSBidirection          VolcengineAgentPlanEndpoint = "tts_bidirection"
	VolcengineAgentPlanASRBigmodel             VolcengineAgentPlanEndpoint = "asr_bigmodel"
	VolcengineAgentPlanASRBigmodelAsync        VolcengineAgentPlanEndpoint = "asr_bigmodel_async"
	VolcengineAgentPlanASRBigmodelNostream     VolcengineAgentPlanEndpoint = "asr_bigmodel_nostream"
	volcengineAgentPlanServerErrorCooldown                                 = 30 * time.Second

	VolcengineAgentPlanSeedreamModel = "doubao-seedream-5.0-lite"
	VolcengineAgentPlanTTSModel      = "seed-tts-2.0"
	VolcengineAgentPlanASRModel      = "volc.seedasr.sauc.duration"

	VolcengineAgentPlanTTSResourceID = "seed-tts-2.0"
	VolcengineAgentPlanASRResourceID = "volc.seedasr.sauc.duration"
)

const (
	volcengineAgentPlanMaxBufferedResponseBytes = 64 << 20
	volcengineAgentPlanMaxBufferedErrorBytes    = 2 << 20
	volcengineAgentPlanMaxObservedLineBytes     = 1 << 20
)

func VolcengineAgentPlanUpstreamURL(endpoint VolcengineAgentPlanEndpoint) (string, error) {
	switch endpoint {
	case VolcengineAgentPlanImagesGenerations:
		return "https://ark.cn-beijing.volces.com/api/plan/v3/images/generations", nil
	case VolcengineAgentPlanTTSUnidirectional:
		return "https://openspeech.bytedance.com/api/v3/plan/tts/unidirectional", nil
	case VolcengineAgentPlanTTSUnidirectionalStream:
		return "wss://openspeech.bytedance.com/api/v3/plan/tts/unidirectional/stream", nil
	case VolcengineAgentPlanTTSBidirection:
		return "wss://openspeech.bytedance.com/api/v3/plan/tts/bidirection", nil
	case VolcengineAgentPlanASRBigmodel:
		return "wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel", nil
	case VolcengineAgentPlanASRBigmodelAsync:
		return "wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_async", nil
	case VolcengineAgentPlanASRBigmodelNostream:
		return "wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_nostream", nil
	default:
		return "", fmt.Errorf("unsupported Volcengine Agent Plan endpoint: %s", endpoint)
	}
}

func (e VolcengineAgentPlanEndpoint) IsWebSocket() bool {
	switch e {
	case VolcengineAgentPlanTTSUnidirectionalStream,
		VolcengineAgentPlanTTSBidirection,
		VolcengineAgentPlanASRBigmodel,
		VolcengineAgentPlanASRBigmodelAsync,
		VolcengineAgentPlanASRBigmodelNostream:
		return true
	default:
		return false
	}
}

func (e VolcengineAgentPlanEndpoint) BillingModel() string {
	switch e {
	case VolcengineAgentPlanImagesGenerations:
		return VolcengineAgentPlanSeedreamModel
	case VolcengineAgentPlanTTSUnidirectional,
		VolcengineAgentPlanTTSUnidirectionalStream,
		VolcengineAgentPlanTTSBidirection:
		return VolcengineAgentPlanTTSModel
	case VolcengineAgentPlanASRBigmodel,
		VolcengineAgentPlanASRBigmodelAsync,
		VolcengineAgentPlanASRBigmodelNostream:
		return VolcengineAgentPlanASRModel
	default:
		return ""
	}
}

func VolcengineAgentPlanWebSocketHeaders(endpoint VolcengineAgentPlanEndpoint, apiKey string) http.Header {
	return volcengineAgentPlanProxyHeaders(nil, endpoint, apiKey, "")
}

func VolcengineAgentPlanHTTPHeaders(endpoint VolcengineAgentPlanEndpoint, apiKey string) http.Header {
	return volcengineAgentPlanProxyHeaders(nil, endpoint, apiKey, "")
}

func volcengineAgentPlanProxyHeaders(source http.Header, endpoint VolcengineAgentPlanEndpoint, apiKey, upstreamModel string) http.Header {
	headers := make(http.Header)
	for key, values := range source {
		if volcengineAgentPlanHopByHopHeader(key) || volcengineAgentPlanPrivateRequestHeader(key) ||
			strings.EqualFold(key, "Authorization") ||
			strings.EqualFold(key, "X-Api-Key") || strings.EqualFold(key, "X-Goog-Api-Key") {
			continue
		}
		headers[key] = append([]string(nil), values...)
	}
	if endpoint == VolcengineAgentPlanImagesGenerations {
		headers.Set("Authorization", "Bearer "+apiKey)
	} else {
		headers.Set("X-Api-Key", apiKey)
		if model := strings.TrimSpace(upstreamModel); model != "" {
			headers.Set("X-Api-Resource-Id", model)
		}
	}
	return headers
}

func volcengineAgentPlanPrivateRequestHeader(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	switch lower {
	case "cookie", "set-cookie", "forwarded", "x-real-ip", "x-client-ip", "true-client-ip",
		"cf-connecting-ip", "x-original-url", "x-rewrite-url", "x-original-host":
		return true
	}
	return strings.HasPrefix(lower, "x-forwarded-") ||
		strings.HasPrefix(lower, "x-envoy-") ||
		strings.HasPrefix(lower, "x-internal-")
}

func volcengineAgentPlanHopByHopHeader(key string) bool {
	switch {
	case strings.EqualFold(key, "Connection"),
		strings.EqualFold(key, "Keep-Alive"),
		strings.EqualFold(key, "Proxy-Authenticate"),
		strings.EqualFold(key, "Proxy-Authorization"),
		strings.EqualFold(key, "TE"),
		strings.EqualFold(key, "Trailer"),
		strings.EqualFold(key, "Transfer-Encoding"),
		strings.EqualFold(key, "Upgrade"),
		strings.EqualFold(key, "Host"),
		strings.EqualFold(key, "Content-Length"),
		strings.HasPrefix(strings.ToLower(key), "sec-websocket-"):
		return true
	default:
		return false
	}
}

type VolcengineAgentPlanUpstreamError struct {
	StatusCode   int
	ProviderCode string
	Body         []byte
	Headers      http.Header
}

func (e *VolcengineAgentPlanUpstreamError) Error() string {
	if e == nil {
		return "volcengine agent plan upstream error"
	}
	return fmt.Sprintf("volcengine agent plan upstream returned HTTP %d", e.StatusCode)
}

func (e *VolcengineAgentPlanUpstreamError) Retryable() bool {
	if e == nil {
		return false
	}
	return e.StatusCode == http.StatusUnauthorized ||
		e.StatusCode == http.StatusForbidden ||
		e.StatusCode == http.StatusRequestTimeout ||
		e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode >= http.StatusInternalServerError
}

func (s *GatewayService) HandleVolcengineAgentPlanUpstreamError(ctx context.Context, account *Account, err error, requestedModel string) {
	if s == nil || s.rateLimitService == nil || account == nil || err == nil {
		return
	}
	var upstreamErr *VolcengineAgentPlanUpstreamError
	if !errors.As(err, &upstreamErr) || upstreamErr == nil {
		return
	}
	stopped := s.rateLimitService.HandleUpstreamError(
		ctx, account, upstreamErr.StatusCode, upstreamErr.Headers, upstreamErr.Body, requestedModel,
	)
	if !stopped && upstreamErr.StatusCode >= http.StatusInternalServerError {
		s.rateLimitService.coolDownVolcengineAgentPlanServerFailure(ctx, account, upstreamErr.StatusCode, upstreamErr.Body)
	}
}

func (s *RateLimitService) coolDownVolcengineAgentPlanServerFailure(ctx context.Context, account *Account, statusCode int, body []byte) {
	if s == nil || s.accountRepo == nil || account == nil {
		return
	}
	now := time.Now()
	until := now.Add(volcengineAgentPlanServerErrorCooldown)
	message := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	reason := fmt.Sprintf("Volcengine Agent Plan upstream HTTP %d: %s", statusCode, message)
	state := &TempUnschedState{
		UntilUnix:       until.Unix(),
		TriggeredAtUnix: now.Unix(),
		StatusCode:      statusCode,
		MatchedKeyword:  "volcengine_agent_plan_server_error",
		RuleIndex:       -1,
		ErrorMessage:    reason,
	}
	s.notifyAccountSchedulingBlocked(account, until, "volcengine_agent_plan_server_error")
	if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, until, reason); err != nil {
		return
	}
	if s.tempUnschedCache != nil {
		_ = s.tempUnschedCache.SetTempUnsched(ctx, account.ID, state)
	}
}

func (s *GatewayService) ValidateVolcengineAgentPlanPricing(
	ctx context.Context,
	apiKey *APIKey,
	endpoint VolcengineAgentPlanEndpoint,
	usageFields ChannelUsageFields,
	upstreamModel string,
) error {
	if s == nil || apiKey == nil || apiKey.GroupID == nil || apiKey.Group == nil || s.resolver == nil {
		return fmt.Errorf("%w: Volcengine Agent Plan channel pricing is unavailable", ErrBillablePricingRequired)
	}
	billingModel := strings.TrimSpace(usageFields.ChannelMappedModel)
	switch usageFields.BillingModelSource {
	case BillingModelSourceRequested:
		billingModel = strings.TrimSpace(usageFields.OriginalModel)
	case BillingModelSourceUpstream:
		billingModel = strings.TrimSpace(upstreamModel)
	}
	candidates := usageBillingModelCandidates(
		billingModel,
		usageFields.ChannelMappedModel,
		usageFields.OriginalModel,
		upstreamModel,
	)
	for _, candidate := range candidates {
		resolved := s.resolveChannelPricing(ctx, candidate, apiKey)
		if resolved == nil || !volcengineAgentPlanPricingModeAllowed(endpoint, resolved.Mode) {
			continue
		}
		if resolved.DefaultPerRequestPrice > 0 {
			return nil
		}
		for _, tier := range resolved.RequestTiers {
			if tier.PerRequestPrice != nil && *tier.PerRequestPrice > 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("%w: Volcengine Agent Plan model=%s requires positive per-request channel pricing", ErrBillablePricingRequired, billingModel)
}

func volcengineAgentPlanPricingModeAllowed(endpoint VolcengineAgentPlanEndpoint, mode BillingMode) bool {
	if endpoint == VolcengineAgentPlanImagesGenerations {
		return mode == BillingModePerRequest || mode == BillingModeImage
	}
	return mode == BillingModePerRequest
}

func appendVolcengineAgentPlanOpsError(
	c *gin.Context,
	account *Account,
	targetURL string,
	statusCode int,
	headers http.Header,
	body []byte,
	kind string,
	reason string,
	cause error,
) {
	if c == nil {
		return
	}
	message := "Volcengine Agent Plan upstream request failed"
	if extracted := strings.TrimSpace(extractUpstreamErrorMessage(body)); extracted != "" {
		message = extracted
	} else if cause != nil {
		message = cause.Error()
	} else if statusCode > 0 {
		message = fmt.Sprintf("Volcengine Agent Plan upstream returned HTTP %d", statusCode)
	}
	detail := ""
	if len(body) > 0 {
		detail = truncateString(string(body), 4096)
	}
	event := OpsUpstreamErrorEvent{
		Passthrough:          true,
		UpstreamStatusCode:   statusCode,
		UpstreamURL:          safeUpstreamURL(targetURL),
		UpstreamRequestID:    volcengineAgentPlanResponseRequestID(headers),
		UpstreamResponseBody: detail,
		Kind:                 kind,
		Reason:               strings.TrimSpace(reason),
		Message:              message,
		Detail:               detail,
	}
	if account != nil {
		event.Platform = account.Platform
		event.AccountID = account.ID
		event.AccountName = account.Name
	}
	appendOpsUpstreamError(c, event)
}

func (s *GatewayService) ForwardVolcengineAgentPlanHTTP(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	endpoint VolcengineAgentPlanEndpoint,
	body []byte,
	upstreamModel string,
) (*ForwardResult, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, fmt.Errorf("http upstream is not configured")
	}
	if account == nil || account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("volcengine agent plan requires an API Key account")
	}
	if endpoint.IsWebSocket() {
		return nil, fmt.Errorf("websocket endpoint cannot be forwarded over HTTP")
	}
	targetURL, err := VolcengineAgentPlanUpstreamURL(endpoint)
	if err != nil {
		return nil, err
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	forwardBody := body
	streamResponse := endpoint == VolcengineAgentPlanTTSUnidirectional
	if endpoint == VolcengineAgentPlanImagesGenerations {
		if strings.TrimSpace(upstreamModel) != "" {
			forwardBody, err = sjson.SetBytes(body, "model", upstreamModel)
			if err != nil {
				return nil, fmt.Errorf("rewrite Volcengine image model: %w", err)
			}
		}
		streamResponse = gjson.GetBytes(forwardBody, "stream").Bool()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(forwardBody))
	if err != nil {
		return nil, err
	}
	requestHeaders := http.Header(nil)
	if c != nil && c.Request != nil {
		requestHeaders = c.Request.Header
	}
	for key, values := range volcengineAgentPlanProxyHeaders(requestHeaders, endpoint, token, upstreamModel) {
		req.Header[key] = append([]string(nil), values...)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.tlsFPProfileService != nil {
		if tlsProfile := s.tlsFPProfileService.ResolveTLSProfile(account); tlsProfile != nil {
			start := time.Now()
			resp, requestErr := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
			return s.finishVolcengineAgentPlanHTTP(c, account, targetURL, resp, requestErr, endpoint, upstreamModel, streamResponse, start)
		}
	}
	start := time.Now()
	resp, requestErr := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	return s.finishVolcengineAgentPlanHTTP(c, account, targetURL, resp, requestErr, endpoint, upstreamModel, streamResponse, start)
}

func (s *GatewayService) finishVolcengineAgentPlanHTTP(
	c *gin.Context,
	account *Account,
	targetURL string,
	resp *http.Response,
	requestErr error,
	endpoint VolcengineAgentPlanEndpoint,
	upstreamModel string,
	streamResponse bool,
	start time.Time,
) (*ForwardResult, error) {
	if requestErr != nil {
		appendVolcengineAgentPlanOpsError(c, account, targetURL, 0, nil, nil, "request_error", "", requestErr)
		return nil, fmt.Errorf("volcengine agent plan upstream request failed: %w", requestErr)
	}
	if resp == nil {
		err := fmt.Errorf("volcengine agent plan upstream returned no response")
		appendVolcengineAgentPlanOpsError(c, account, targetURL, 0, nil, nil, "request_error", "", err)
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	providerCode := strings.TrimSpace(resp.Header.Get("X-Api-Status-Code"))
	providerFailed := providerCode != "" && !volcengineAgentPlanSuccessCode(providerCode)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, streamed, err := s.readOrStreamVolcengineAgentPlanErrorResponse(c, resp)
		if err != nil {
			appendVolcengineAgentPlanOpsError(c, account, targetURL, resp.StatusCode, resp.Header, nil, "response_read_error", providerCode, err)
			return nil, err
		}
		appendVolcengineAgentPlanOpsError(c, account, targetURL, resp.StatusCode, resp.Header, body, "http_error", providerCode, nil)
		if streamed {
			return nil, fmt.Errorf("volcengine agent plan upstream returned HTTP %d", resp.StatusCode)
		}
		return nil, &VolcengineAgentPlanUpstreamError{
			StatusCode:   resp.StatusCode,
			ProviderCode: providerCode,
			Body:         body,
			Headers:      resp.Header.Clone(),
		}
	}
	if streamResponse || endpoint == VolcengineAgentPlanImagesGenerations {
		stats, err := s.streamVolcengineAgentPlanHTTPResponse(c, resp, endpoint, streamResponse)
		if err != nil {
			appendVolcengineAgentPlanOpsError(c, account, targetURL, resp.StatusCode, resp.Header, nil, "stream_error", providerCode, err)
			return nil, err
		}
		if providerFailed || !stats.Successful(endpoint) {
			err := fmt.Errorf("volcengine agent plan response did not complete successfully")
			appendVolcengineAgentPlanOpsError(c, account, targetURL, resp.StatusCode, resp.Header, []byte(stats.FailureMessage), "business_error", providerCode, err)
			return nil, err
		}
		model := strings.TrimSpace(upstreamModel)
		result := &ForwardResult{
			RequestID:     volcengineAgentPlanResponseRequestID(resp.Header),
			Model:         model,
			UpstreamModel: model,
			RequestCount:  1,
			Stream:        streamResponse,
			Duration:      time.Since(start),
		}
		if endpoint == VolcengineAgentPlanImagesGenerations {
			result.ImageCount = stats.ImageCount
			if result.ImageCount <= 0 {
				result.ImageCount = 1
			}
		}
		return result, nil
	}

	body, err := readVolcengineAgentPlanResponseBody(resp.Body)
	if err != nil {
		appendVolcengineAgentPlanOpsError(c, account, targetURL, resp.StatusCode, resp.Header, nil, "response_read_error", providerCode, err)
		return nil, err
	}

	stats := parseVolcengineAgentPlanResponseStats(endpoint, body)

	if c != nil {
		s.WriteVolcengineAgentPlanResponseHeaders(c.Writer.Header(), resp.Header)
		contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Data(resp.StatusCode, contentType, body)
	}
	if providerFailed || (endpoint == VolcengineAgentPlanImagesGenerations && !stats.Successful(endpoint)) {
		err := fmt.Errorf("volcengine agent plan response did not complete successfully")
		appendVolcengineAgentPlanOpsError(c, account, targetURL, resp.StatusCode, resp.Header, body, "business_error", providerCode, err)
		return nil, err
	}
	model := strings.TrimSpace(upstreamModel)
	if responseModel := strings.TrimSpace(gjson.GetBytes(body, "model").String()); responseModel != "" {
		model = responseModel
	}
	result := &ForwardResult{
		RequestID:     volcengineAgentPlanResponseRequestID(resp.Header),
		Model:         model,
		UpstreamModel: model,
		RequestCount:  1,
		Duration:      time.Since(start),
	}
	if endpoint == VolcengineAgentPlanImagesGenerations {
		result.ImageCount = stats.ImageCount
		if result.ImageCount <= 0 {
			result.ImageCount = 1
		}
	}
	return result, nil
}

func (s *GatewayService) streamVolcengineAgentPlanHTTPResponse(c *gin.Context, resp *http.Response, endpoint VolcengineAgentPlanEndpoint, flush bool) (volcengineAgentPlanResponseStats, error) {
	observer := newVolcengineAgentPlanResponseObserver(endpoint)
	if c == nil {
		_, err := io.Copy(observer, resp.Body)
		return observer.Stats(), err
	}
	s.WriteVolcengineAgentPlanResponseHeaders(c.Writer.Header(), resp.Header)
	if strings.TrimSpace(c.Writer.Header().Get("Content-Type")) == "" {
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
	}
	if flush {
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")
	}
	c.Status(resp.StatusCode)

	flusher, _ := c.Writer.(http.Flusher)
	writer := io.Writer(c.Writer)
	if flush && flusher != nil {
		writer = &volcengineAgentPlanFlushWriter{writer: c.Writer, flusher: flusher}
	}
	if _, err := io.CopyBuffer(writer, io.TeeReader(resp.Body, observer), make([]byte, 32<<10)); err != nil {
		return volcengineAgentPlanResponseStats{}, fmt.Errorf("stream Volcengine Agent Plan response: %w", err)
	}
	return observer.Stats(), nil
}

type volcengineAgentPlanFlushWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func (w *volcengineAgentPlanFlushWriter) Write(payload []byte) (int, error) {
	written, err := w.writer.Write(payload)
	if written > 0 {
		w.flusher.Flush()
	}
	return written, err
}

type volcengineAgentPlanResponseStats struct {
	ImageCount     int
	Completed      bool
	Failed         bool
	ImageEvents    int
	FailureMessage string
}

type volcengineAgentPlanResponseObserver struct {
	endpoint     VolcengineAgentPlanEndpoint
	pending      []byte
	stats        volcengineAgentPlanResponseStats
	imageMode    byte
	droppingLine bool
	imageJSON    *volcengineAgentPlanImageJSONObserver
}

func newVolcengineAgentPlanResponseObserver(endpoint VolcengineAgentPlanEndpoint) *volcengineAgentPlanResponseObserver {
	return &volcengineAgentPlanResponseObserver{endpoint: endpoint}
}

func (o *volcengineAgentPlanResponseObserver) Write(payload []byte) (int, error) {
	if o == nil {
		return len(payload), nil
	}
	if o.endpoint == VolcengineAgentPlanTTSUnidirectional {
		o.pending = append(o.pending, payload...)
		o.consumeTTSJSON()
		return len(payload), nil
	}
	if o.endpoint == VolcengineAgentPlanImagesGenerations {
		o.observeImagePayload(payload)
		return len(payload), nil
	}
	return len(payload), nil
}

func (o *volcengineAgentPlanResponseObserver) Stats() volcengineAgentPlanResponseStats {
	if o == nil {
		return volcengineAgentPlanResponseStats{}
	}
	if o.endpoint == VolcengineAgentPlanTTSUnidirectional {
		o.consumeTTSJSON()
		if len(bytes.TrimSpace(o.pending)) > 0 {
			o.stats.Failed = true
		}
	} else if o.imageMode == 'j' && o.imageJSON != nil {
		o.stats.ImageCount = o.imageJSON.imageCount
		o.stats.Completed = o.imageJSON.completed
		o.stats.Failed = o.imageJSON.failed
	} else {
		if !o.droppingLine {
			o.observeLine(o.pending)
		}
	}
	o.pending = nil
	return o.stats
}

func (o *volcengineAgentPlanResponseObserver) observeImagePayload(payload []byte) {
	if o.imageMode == 0 {
		o.pending = append(o.pending, payload...)
		trimmed := bytes.TrimSpace(o.pending)
		if len(trimmed) == 0 {
			return
		}
		if trimmed[0] == '{' {
			o.imageMode = 'j'
			o.imageJSON = &volcengineAgentPlanImageJSONObserver{dataArrayDepth: -1, itemObjectDepth: -1}
			o.imageJSON.Write(o.pending)
			o.pending = nil
			return
		}
		o.imageMode = 'l'
		payload = o.pending
		o.pending = nil
	}
	if o.imageMode == 'j' {
		o.imageJSON.Write(payload)
		return
	}
	o.observeImageLines(payload)
}

func (o *volcengineAgentPlanResponseObserver) observeImageLines(payload []byte) {
	for len(payload) > 0 {
		if o.droppingLine {
			index := bytes.IndexByte(payload, '\n')
			if index < 0 {
				return
			}
			o.droppingLine = false
			payload = payload[index+1:]
			continue
		}
		index := bytes.IndexByte(payload, '\n')
		if index >= 0 {
			if len(o.pending)+index <= volcengineAgentPlanMaxObservedLineBytes {
				o.pending = append(o.pending, payload[:index]...)
				o.observeLine(o.pending)
			}
			o.pending = nil
			payload = payload[index+1:]
			continue
		}
		if len(o.pending)+len(payload) > volcengineAgentPlanMaxObservedLineBytes {
			o.pending = nil
			o.droppingLine = true
			return
		}
		o.pending = append(o.pending, payload...)
		return
	}
}

type volcengineAgentPlanImageJSONObserver struct {
	stack           []byte
	inString        bool
	escaped         bool
	stringIsValue   bool
	stringIsTarget  bool
	stringHasValue  bool
	stringBuffer    []byte
	candidate       string
	currentKey      string
	expectValue     bool
	dataArrayDepth  int
	itemObjectDepth int
	itemHasImage    bool
	imageCount      int
	completed       bool
	failed          bool
}

func (o *volcengineAgentPlanImageJSONObserver) Write(payload []byte) {
	if o == nil {
		return
	}
	for _, current := range payload {
		if o.inString {
			if o.escaped {
				o.escaped = false
				if o.stringIsTarget {
					o.stringHasValue = true
				}
				continue
			}
			if current == '\\' {
				o.escaped = true
				continue
			}
			if current == '"' {
				o.finishString()
				continue
			}
			if o.stringIsTarget {
				o.stringHasValue = true
			} else if !o.stringIsValue && len(o.stringBuffer) < 128 {
				o.stringBuffer = append(o.stringBuffer, current)
			}
			continue
		}

		switch current {
		case ' ', '\t', '\r', '\n':
			continue
		case '"':
			o.inString = true
			o.stringIsValue = o.expectValue
			o.stringIsTarget = o.expectValue && o.itemObjectDepth > 0 && len(o.stack) == o.itemObjectDepth &&
				(o.currentKey == "url" || o.currentKey == "b64_json")
			o.stringHasValue = false
			o.stringBuffer = o.stringBuffer[:0]
		case ':':
			o.currentKey = o.candidate
			o.candidate = ""
			o.expectValue = true
			if len(o.stack) == 1 && o.currentKey == "error" {
				o.failed = true
			}
		case '{':
			if o.dataArrayDepth > 0 && len(o.stack) == o.dataArrayDepth && o.stack[len(o.stack)-1] == '[' {
				o.itemObjectDepth = len(o.stack) + 1
				o.itemHasImage = false
			}
			o.stack = append(o.stack, current)
			o.clearValueState()
		case '[':
			if o.expectValue && o.currentKey == "data" && len(o.stack) == 1 && o.stack[0] == '{' {
				o.dataArrayDepth = len(o.stack) + 1
			}
			o.stack = append(o.stack, current)
			o.clearValueState()
		case '}':
			if len(o.stack) == o.itemObjectDepth && o.itemObjectDepth > 0 {
				if o.itemHasImage {
					o.imageCount++
				}
				o.itemObjectDepth = -1
				o.itemHasImage = false
			}
			if len(o.stack) == 1 && o.stack[0] == '{' {
				o.completed = true
			}
			o.pop('}')
			o.clearValueState()
		case ']':
			if len(o.stack) == o.dataArrayDepth && o.dataArrayDepth > 0 {
				o.dataArrayDepth = -1
			}
			o.pop(']')
			o.clearValueState()
		case ',':
			o.clearValueState()
		default:
			if o.expectValue {
				o.clearValueState()
			}
		}
	}
}

func (o *volcengineAgentPlanImageJSONObserver) finishString() {
	o.inString = false
	if o.stringIsTarget && o.stringHasValue {
		o.itemHasImage = true
	}
	if o.stringIsValue {
		o.clearValueState()
	} else {
		o.candidate = string(o.stringBuffer)
	}
	o.stringIsValue = false
	o.stringIsTarget = false
	o.stringHasValue = false
	o.stringBuffer = o.stringBuffer[:0]
}

func (o *volcengineAgentPlanImageJSONObserver) clearValueState() {
	o.expectValue = false
	o.currentKey = ""
	o.candidate = ""
}

func (o *volcengineAgentPlanImageJSONObserver) pop(closing byte) {
	if len(o.stack) == 0 {
		return
	}
	opening := o.stack[len(o.stack)-1]
	if (closing == '}' && opening != '{') || (closing == ']' && opening != '[') {
		o.failed = true
		return
	}
	o.stack = o.stack[:len(o.stack)-1]
}

func (o *volcengineAgentPlanResponseObserver) observeLine(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	if bytes.HasPrefix(bytes.ToLower(line), []byte("data:")) {
		line = bytes.TrimSpace(line[len("data:"):])
	}
	if !json.Valid(line) {
		return
	}
	o.observeImageJSON(line)
}

func (o *volcengineAgentPlanResponseObserver) consumeTTSJSON() {
	for {
		trimmed := bytes.TrimLeft(o.pending, " \t\r\n")
		if len(trimmed) == 0 {
			o.pending = nil
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				o.pending = trimmed
				return
			}
			o.pending = trimmed
			return
		}
		consumed := int(decoder.InputOffset())
		if consumed <= 0 || consumed > len(trimmed) {
			o.pending = trimmed
			return
		}
		o.observeTTSJSON(raw)
		o.pending = trimmed[consumed:]
	}
}

func (o *volcengineAgentPlanResponseObserver) observeTTSJSON(body []byte) {
	code := strings.TrimSpace(gjson.GetBytes(body, "code").String())
	if code == "" {
		return
	}
	if code == "20000000" {
		o.stats.Completed = true
		return
	}
	if code != "0" && code != "200" {
		o.stats.Failed = true
		o.stats.FailureMessage = strings.TrimSpace(gjson.GetBytes(body, "message").String())
	}
}

func (o *volcengineAgentPlanResponseObserver) observeImageJSON(body []byte) {
	typeName := strings.TrimSpace(gjson.GetBytes(body, "type").String())
	switch typeName {
	case "image_generation.partial_succeeded":
		o.stats.ImageEvents++
	case "image_generation.completed":
		o.stats.Completed = true
		generatedImages := gjson.GetBytes(body, "usage.generated_images")
		if generatedImages.Exists() {
			o.stats.ImageCount = int(generatedImages.Int())
		} else {
			o.stats.ImageCount = o.stats.ImageEvents
		}
		if errorNode := gjson.GetBytes(body, "error"); errorNode.Exists() && o.stats.ImageCount == 0 {
			o.stats.Failed = true
			o.stats.FailureMessage = errorNode.Raw
		}
	case "image_generation.partial_failed":
		// A failed image does not fail the whole group; the completed event supplies
		// the final successful image count.
	default:
		data := gjson.GetBytes(body, "data")
		if data.IsArray() {
			count := 0
			for _, item := range data.Array() {
				if strings.TrimSpace(item.Get("url").String()) != "" ||
					strings.TrimSpace(item.Get("b64_json").String()) != "" {
					count++
				}
			}
			o.stats.ImageCount = count
			o.stats.Completed = true
		}
		if errorNode := gjson.GetBytes(body, "error"); errorNode.Exists() {
			o.stats.Failed = true
			o.stats.FailureMessage = errorNode.Raw
		}
	}
}

func (o *volcengineAgentPlanResponseObserver) Successful(endpoint VolcengineAgentPlanEndpoint) bool {
	if o == nil {
		return false
	}
	return o.stats.Successful(endpoint)
}

func (s volcengineAgentPlanResponseStats) Successful(endpoint VolcengineAgentPlanEndpoint) bool {
	if s.Failed || !s.Completed {
		return false
	}
	if endpoint == VolcengineAgentPlanImagesGenerations {
		return s.ImageCount > 0
	}
	return true
}

func parseVolcengineAgentPlanResponseStats(endpoint VolcengineAgentPlanEndpoint, body []byte) volcengineAgentPlanResponseStats {
	observer := newVolcengineAgentPlanResponseObserver(endpoint)
	_, _ = observer.Write(body)
	return observer.Stats()
}

func volcengineAgentPlanWebSocketResponseHeaders(headers http.Header) http.Header {
	allowed := make(http.Header)
	copyVolcengineAgentPlanDiagnosticHeaders(allowed, headers)
	return allowed
}

func copyVolcengineAgentPlanDiagnosticHeaders(dst, src http.Header) {
	for _, key := range []string{"X-Api-Status-Code", "X-Api-Message", "X-Tt-Logid", "X-Api-Connect-Id", "X-Request-Id"} {
		values := src.Values(key)
		if len(values) == 0 {
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// WriteVolcengineAgentPlanResponseHeaders applies the shared response-header
// policy and then restores the provider diagnostics required for troubleshooting.
func (s *GatewayService) WriteVolcengineAgentPlanResponseHeaders(dst, src http.Header) {
	if dst == nil || src == nil {
		return
	}
	var filter *responseheaders.CompiledHeaderFilter
	if s != nil {
		filter = s.responseHeaderFilter
	}
	responseheaders.WriteFilteredHeaders(dst, src, filter)
	copyVolcengineAgentPlanDiagnosticHeaders(dst, src)
}

func volcengineAgentPlanResponseRequestID(headers http.Header) string {
	if requestID := strings.TrimSpace(headers.Get("X-Request-Id")); requestID != "" {
		return requestID
	}
	return strings.TrimSpace(headers.Get("X-Tt-Logid"))
}

func readVolcengineAgentPlanResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, volcengineAgentPlanMaxBufferedResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read Volcengine Agent Plan response: %w", err)
	}
	if len(data) > volcengineAgentPlanMaxBufferedResponseBytes {
		return nil, fmt.Errorf("volcengine agent plan buffered response exceeds %d bytes", volcengineAgentPlanMaxBufferedResponseBytes)
	}
	return data, nil
}

func (s *GatewayService) readOrStreamVolcengineAgentPlanErrorResponse(c *gin.Context, resp *http.Response) ([]byte, bool, error) {
	limited := io.LimitReader(resp.Body, volcengineAgentPlanMaxBufferedErrorBytes+1)
	buffered, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, fmt.Errorf("read Volcengine Agent Plan error response: %w", err)
	}
	if len(buffered) <= volcengineAgentPlanMaxBufferedErrorBytes {
		return buffered, false, nil
	}
	preview := append([]byte(nil), buffered[:min(len(buffered), 4096)]...)
	if c == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return preview, false, fmt.Errorf("volcengine Agent Plan error response requires streaming")
	}

	s.WriteVolcengineAgentPlanResponseHeaders(c.Writer.Header(), resp.Header)
	if strings.TrimSpace(c.Writer.Header().Get("Content-Type")) == "" {
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
	}
	c.Status(resp.StatusCode)
	if _, err := c.Writer.Write(buffered); err != nil {
		return preview, true, fmt.Errorf("write Volcengine Agent Plan error response: %w", err)
	}
	if _, err := io.CopyBuffer(c.Writer, resp.Body, make([]byte, 32<<10)); err != nil {
		return preview, true, fmt.Errorf("stream Volcengine Agent Plan error response: %w", err)
	}
	return preview, true, nil
}

func (s *GatewayService) ProxyVolcengineAgentPlanWebSocket(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	endpoint VolcengineAgentPlanEndpoint,
	upstreamModel string,
) (*ForwardResult, error) {
	if account == nil || account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("volcengine agent plan requires an API Key account")
	}
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("missing WebSocket request context")
	}
	if !endpoint.IsWebSocket() {
		return nil, fmt.Errorf("http endpoint cannot be forwarded over websocket")
	}
	targetURL, err := VolcengineAgentPlanUpstreamURL(endpoint)
	if err != nil {
		return nil, err
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{HandshakeTimeout: 20 * time.Second}
	if account.ProxyID != nil && account.Proxy != nil {
		parsedProxy, parseErr := url.Parse(account.Proxy.URL())
		if parseErr != nil {
			return nil, fmt.Errorf("invalid account proxy: %w", parseErr)
		}
		dialer.Proxy = http.ProxyURL(parsedProxy)
	}
	requestHeaders := http.Header(nil)
	if c.Request != nil {
		requestHeaders = c.Request.Header
	}
	upstreamConn, upstreamResp, err := dialer.DialContext(ctx, targetURL, volcengineAgentPlanProxyHeaders(requestHeaders, endpoint, token, upstreamModel))
	if err != nil {
		if upstreamResp != nil {
			defer func() { _ = upstreamResp.Body.Close() }()
			body, _ := io.ReadAll(io.LimitReader(upstreamResp.Body, 2<<20))
			appendVolcengineAgentPlanOpsError(c, account, targetURL, upstreamResp.StatusCode, upstreamResp.Header, body, "http_error", "", err)
			return nil, &VolcengineAgentPlanUpstreamError{
				StatusCode: upstreamResp.StatusCode,
				Body:       body,
				Headers:    upstreamResp.Header.Clone(),
			}
		}
		appendVolcengineAgentPlanOpsError(c, account, targetURL, 0, nil, nil, "request_error", "", err)
		return nil, fmt.Errorf("connect Volcengine Agent Plan WebSocket: %w", err)
	}
	defer func() { _ = upstreamConn.Close() }()
	if upstreamResp != nil && upstreamResp.Body != nil {
		defer func() { _ = upstreamResp.Body.Close() }()
	}

	clientResponseHeaders := make(http.Header)
	if upstreamResp != nil {
		clientResponseHeaders = volcengineAgentPlanWebSocketResponseHeaders(upstreamResp.Header)
	}
	clientConn, err := (&websocket.Upgrader{
		ReadBufferSize:  32 << 10,
		WriteBufferSize: 32 << 10,
	}).Upgrade(c.Writer, c.Request, clientResponseHeaders)
	if err != nil {
		return nil, fmt.Errorf("upgrade client WebSocket: %w", err)
	}
	defer func() { _ = clientConn.Close() }()

	start := time.Now()
	tracker := newVolcengineAgentPlanWSUsageTracker(endpoint)
	stopOnCancel := context.AfterFunc(ctx, func() {
		_ = clientConn.Close()
		_ = upstreamConn.Close()
	})
	defer stopOnCancel()

	errCh := make(chan error, 2)
	go proxyVolcengineAgentPlanFrames(clientConn, upstreamConn, tracker.ObserveClientMessage, errCh)
	go proxyVolcengineAgentPlanFrames(upstreamConn, clientConn, tracker.ObserveUpstreamMessage, errCh)

	firstErr := <-errCh
	_ = clientConn.Close()
	_ = upstreamConn.Close()
	select {
	case <-errCh:
	case <-time.After(time.Second):
	}

	requestCount := tracker.RequestCount()
	if tracker.Failed() || requestCount == 0 {
		message := tracker.FailureMessage()
		if message == "" {
			message = "websocket closed before a successful final result"
		}
		failure := fmt.Errorf("volcengine agent plan %s: %w", message, firstErr)
		var upstreamHeaders http.Header
		if upstreamResp != nil {
			upstreamHeaders = upstreamResp.Header
		}
		appendVolcengineAgentPlanOpsError(c, account, targetURL, http.StatusSwitchingProtocols, upstreamHeaders, []byte(message), "websocket_error", "", failure)
		return nil, failure
	}
	requestID := ""
	if upstreamResp != nil {
		requestID = volcengineAgentPlanResponseRequestID(upstreamResp.Header)
	}
	return &ForwardResult{
		RequestID:     requestID,
		Model:         upstreamModel,
		UpstreamModel: upstreamModel,
		RequestCount:  requestCount,
		Stream:        true,
		Duration:      time.Since(start),
	}, nil
}

func proxyVolcengineAgentPlanFrames(
	source *websocket.Conn,
	destination *websocket.Conn,
	observe func([]byte),
	errCh chan<- error,
) {
	for {
		messageType, payload, err := source.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				_ = destination.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(closeErr.Code, closeErr.Text),
					time.Now().Add(time.Second),
				)
			}
			errCh <- err
			return
		}
		if observe != nil {
			observe(payload)
		}
		if err := destination.WriteMessage(messageType, payload); err != nil {
			errCh <- err
			return
		}
	}
}

type volcengineAgentPlanWSUsageTracker struct {
	mu              sync.Mutex
	endpoint        VolcengineAgentPlanEndpoint
	pendingSessions int
	requestCount    int
	failed          bool
	failureMessage  string
}

func newVolcengineAgentPlanWSUsageTracker(endpoint VolcengineAgentPlanEndpoint) *volcengineAgentPlanWSUsageTracker {
	return &volcengineAgentPlanWSUsageTracker{endpoint: endpoint}
}

func (t *volcengineAgentPlanWSUsageTracker) ObserveClientMessage(payload []byte) {
	if t == nil || t.endpoint != VolcengineAgentPlanTTSBidirection || !volcengineAgentPlanStartSession(payload) {
		return
	}
	t.mu.Lock()
	t.pendingSessions++
	t.mu.Unlock()
}

func (t *volcengineAgentPlanWSUsageTracker) ObserveUpstreamMessage(payload []byte) {
	if t == nil {
		return
	}
	observation := observeVolcengineAgentPlanWebSocketMessage(payload)
	t.mu.Lock()
	defer t.mu.Unlock()
	if observation.Failed {
		t.failed = true
		if observation.Message != "" {
			t.failureMessage = observation.Message
		}
		return
	}
	if t.endpoint == VolcengineAgentPlanTTSBidirection {
		if observation.Event == volcengineAgentPlanEventSessionFinished && t.pendingSessions > 0 {
			t.pendingSessions--
			t.requestCount++
		}
		return
	}
	if observation.Final && t.requestCount == 0 {
		t.requestCount = 1
	}
}

func (t *volcengineAgentPlanWSUsageTracker) RequestCount() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failed {
		return 0
	}
	return t.requestCount
}

func (t *volcengineAgentPlanWSUsageTracker) Failed() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

func (t *volcengineAgentPlanWSUsageTracker) FailureMessage() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failureMessage
}

func volcengineAgentPlanStartSession(payload []byte) bool {
	observation := observeVolcengineAgentPlanWebSocketMessage(payload)
	if observation.Event == volcengineAgentPlanEventStartSession {
		return true
	}
	decoded := strings.ToLower(string(decodeVolcengineAgentPlanPayload(payload)))
	return strings.Contains(decoded, "startsession") || strings.Contains(decoded, "start_session")
}

const (
	volcengineAgentPlanMessageTypeFullServerResponse = 0x9
	volcengineAgentPlanMessageTypeAudioResponse      = 0xb
	volcengineAgentPlanMessageTypeError              = 0xf
	volcengineAgentPlanEventStartSession             = 100
	volcengineAgentPlanEventSessionFinished          = 152
	volcengineAgentPlanEventSessionFailed            = 153
)

type volcengineAgentPlanWSObservation struct {
	Event   int32
	Final   bool
	Failed  bool
	Message string
}

type volcengineAgentPlanBinaryFrame struct {
	messageType byte
	flags       byte
	event       int32
	sequence    int32
	hasEvent    bool
	hasSequence bool
	errorCode   uint32
	compression byte
	payload     []byte
}

func observeVolcengineAgentPlanWebSocketMessage(payload []byte) volcengineAgentPlanWSObservation {
	if frame, ok := parseVolcengineAgentPlanBinaryFrame(payload); ok {
		observation := volcengineAgentPlanWSObservation{Event: frame.event}
		if frame.messageType == volcengineAgentPlanMessageTypeError {
			observation.Failed = true
			observation.Message = fmt.Sprintf("upstream websocket error code %d", frame.errorCode)
			if detail := strings.TrimSpace(string(decodeVolcengineAgentPlanFramePayload(frame))); detail != "" {
				observation.Message += ": " + sanitizeUpstreamErrorMessage(truncateString(detail, 2048))
			}
			return observation
		}
		if frame.hasEvent {
			switch frame.event {
			case volcengineAgentPlanEventSessionFinished:
				observation.Final = true
			case volcengineAgentPlanEventSessionFailed, 51:
				observation.Failed = true
				observation.Message = fmt.Sprintf("upstream websocket event %d", frame.event)
			}
			return observation
		}
		if frame.messageType == volcengineAgentPlanMessageTypeFullServerResponse ||
			frame.messageType == volcengineAgentPlanMessageTypeAudioResponse {
			observation.Final = frame.flags&0x2 != 0 || (frame.hasSequence && frame.sequence < 0)
			decoded := decodeVolcengineAgentPlanFramePayload(frame)
			failed, message := volcengineAgentPlanJSONFailure(decoded)
			observation.Failed = failed
			observation.Message = message
			if observation.Failed {
				observation.Final = false
			}
		}
		return observation
	}

	decoded := decodeVolcengineAgentPlanPayload(payload)
	failed, message := volcengineAgentPlanJSONFailure(decoded)
	observation := volcengineAgentPlanWSObservation{Failed: failed, Message: message}
	if len(bytes.TrimSpace(decoded)) == 0 || !json.Valid(decoded) {
		return observation
	}
	event := gjson.GetBytes(decoded, "event")
	if event.Type == gjson.Number {
		observation.Event = int32(event.Int())
	} else {
		switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(event.String()), "_", "")) {
		case "startsession":
			observation.Event = volcengineAgentPlanEventStartSession
		case "sessionfinished":
			observation.Event = volcengineAgentPlanEventSessionFinished
		case "sessionfailed", "connectionfailed":
			observation.Event = volcengineAgentPlanEventSessionFailed
		}
	}
	if observation.Event == volcengineAgentPlanEventSessionFinished {
		observation.Final = true
	}
	if observation.Event == volcengineAgentPlanEventSessionFailed {
		observation.Failed = true
		observation.Final = false
	}
	return observation
}

func parseVolcengineAgentPlanBinaryFrame(payload []byte) (volcengineAgentPlanBinaryFrame, bool) {
	if len(payload) < 4 || payload[0]>>4 != 1 {
		return volcengineAgentPlanBinaryFrame{}, false
	}
	headerSize := int(payload[0]&0x0f) * 4
	if headerSize < 4 || headerSize > len(payload) {
		return volcengineAgentPlanBinaryFrame{}, false
	}
	frame := volcengineAgentPlanBinaryFrame{
		messageType: payload[1] >> 4,
		flags:       payload[1] & 0x0f,
		compression: payload[2] & 0x0f,
	}
	offset := headerSize
	if frame.flags&0x4 != 0 {
		if offset+4 > len(payload) {
			return volcengineAgentPlanBinaryFrame{}, false
		}
		frame.event = int32(binary.BigEndian.Uint32(payload[offset : offset+4]))
		frame.hasEvent = true
		offset += 4
		frame.payload = payload[offset:]
		return frame, true
	}
	if frame.messageType == volcengineAgentPlanMessageTypeError {
		if offset+8 > len(payload) {
			return volcengineAgentPlanBinaryFrame{}, false
		}
		frame.errorCode = binary.BigEndian.Uint32(payload[offset : offset+4])
		offset += 4
		payloadSize := int(binary.BigEndian.Uint32(payload[offset : offset+4]))
		offset += 4
		if payloadSize < 0 || offset+payloadSize > len(payload) {
			return volcengineAgentPlanBinaryFrame{}, false
		}
		frame.payload = payload[offset : offset+payloadSize]
		return frame, true
	}
	if frame.flags&0x1 != 0 {
		if offset+4 > len(payload) {
			return volcengineAgentPlanBinaryFrame{}, false
		}
		frame.sequence = int32(binary.BigEndian.Uint32(payload[offset : offset+4]))
		frame.hasSequence = true
		offset += 4
	}
	if offset+4 > len(payload) {
		frame.payload = payload[offset:]
		return frame, true
	}
	payloadSize := int(binary.BigEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	if payloadSize < 0 || offset+payloadSize > len(payload) {
		return volcengineAgentPlanBinaryFrame{}, false
	}
	frame.payload = payload[offset : offset+payloadSize]
	return frame, true
}

func decodeVolcengineAgentPlanFramePayload(frame volcengineAgentPlanBinaryFrame) []byte {
	if frame.compression != 1 || len(frame.payload) == 0 {
		return frame.payload
	}
	reader, err := gzip.NewReader(bytes.NewReader(frame.payload))
	if err != nil {
		return frame.payload
	}
	decoded, readErr := io.ReadAll(io.LimitReader(reader, 4<<20))
	_ = reader.Close()
	if readErr != nil {
		return frame.payload
	}
	return decoded
}

func volcengineAgentPlanJSONFailure(payload []byte) (bool, string) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return false, ""
	}
	if errorNode := gjson.GetBytes(trimmed, "error"); errorNode.Exists() {
		return true, strings.TrimSpace(errorNode.Raw)
	}
	if strings.EqualFold(strings.TrimSpace(gjson.GetBytes(trimmed, "status").String()), "failed") {
		return true, strings.TrimSpace(gjson.GetBytes(trimmed, "message").String())
	}
	if code := gjson.GetBytes(trimmed, "code"); code.Exists() {
		value := code.Value()
		if !volcengineAgentPlanSuccessValue(value) {
			message := strings.TrimSpace(gjson.GetBytes(trimmed, "message").String())
			if message == "" {
				message = "upstream websocket business error"
			}
			return true, message
		}
	}
	return false, ""
}

func volcengineAgentPlanSuccessValue(code any) bool {
	switch value := code.(type) {
	case float64:
		return value == 0 || value == 200 || value == 20000000
	case string:
		return volcengineAgentPlanSuccessCode(value)
	default:
		return false
	}
}

func volcengineAgentPlanSuccessCode(code string) bool {
	switch strings.TrimSpace(code) {
	case "0", "200", "20000000":
		return true
	default:
		return false
	}
}

func decodeVolcengineAgentPlanPayload(payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return nil
	}
	if json.Valid(trimmed) {
		return trimmed
	}
	for index := 0; index+1 < len(payload); index++ {
		if payload[index] == 0x1f && payload[index+1] == 0x8b {
			reader, err := gzip.NewReader(bytes.NewReader(payload[index:]))
			if err != nil {
				continue
			}
			decoded, readErr := io.ReadAll(io.LimitReader(reader, 4<<20))
			_ = reader.Close()
			if readErr == nil && len(decoded) > 0 {
				return decoded
			}
		}
	}
	if index := bytes.IndexByte(payload, '{'); index >= 0 {
		candidate := bytes.TrimSpace(payload[index:])
		if json.Valid(candidate) {
			return candidate
		}
	}
	return payload
}
