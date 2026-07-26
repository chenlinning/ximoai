package service

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type VolcengineAgentPlanEndpoint string

const (
	VolcengineAgentPlanImagesGenerations       VolcengineAgentPlanEndpoint = "images_generations"
	VolcengineAgentPlanTTSUnidirectional       VolcengineAgentPlanEndpoint = "tts_unidirectional"
	VolcengineAgentPlanTTSUnidirectionalStream VolcengineAgentPlanEndpoint = "tts_unidirectional_stream"
	VolcengineAgentPlanTTSBidirection          VolcengineAgentPlanEndpoint = "tts_bidirection"
	VolcengineAgentPlanASRBigmodelAsync        VolcengineAgentPlanEndpoint = "asr_bigmodel_async"
	VolcengineAgentPlanASRBigmodelNostream     VolcengineAgentPlanEndpoint = "asr_bigmodel_nostream"

	VolcengineAgentPlanSeedreamModel = "doubao-seedream-5.0-lite"
	VolcengineAgentPlanTTSModel      = "seed-tts-2.0"
	VolcengineAgentPlanASRModel      = "volc.seedasr.sauc.duration"

	VolcengineAgentPlanTTSResourceID = "seed-tts-2.0"
	VolcengineAgentPlanASRResourceID = "volc.seedasr.sauc.duration"
)

const volcengineAgentPlanMaxHTTPResponseBytes = 64 << 20

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
	case VolcengineAgentPlanASRBigmodelAsync,
		VolcengineAgentPlanASRBigmodelNostream:
		return VolcengineAgentPlanASRModel
	default:
		return ""
	}
}

func VolcengineAgentPlanWebSocketHeaders(endpoint VolcengineAgentPlanEndpoint, apiKey string) http.Header {
	headers := make(http.Header)
	headers.Set("X-Api-Key", apiKey)
	headers.Set("X-Api-Connect-Id", newVolcengineAgentPlanRequestID())
	switch endpoint {
	case VolcengineAgentPlanTTSUnidirectionalStream,
		VolcengineAgentPlanTTSBidirection:
		headers.Set("X-Api-Resource-Id", VolcengineAgentPlanTTSResourceID)
		headers.Set("X-Control-Require-Usage-Tokens-Return", "*")
	case VolcengineAgentPlanASRBigmodelAsync, VolcengineAgentPlanASRBigmodelNostream:
		headers.Set("X-Api-Resource-Id", VolcengineAgentPlanASRResourceID)
		headers.Set("X-Api-Request-Id", newVolcengineAgentPlanRequestID())
		headers.Set("X-Api-Sequence", "-1")
	}
	return headers
}

func VolcengineAgentPlanHTTPHeaders(endpoint VolcengineAgentPlanEndpoint, apiKey string) http.Header {
	headers := make(http.Header)
	if endpoint != VolcengineAgentPlanTTSUnidirectional {
		return headers
	}
	headers.Set("X-Api-Key", apiKey)
	headers.Set("X-Api-Resource-Id", VolcengineAgentPlanTTSResourceID)
	headers.Set("X-Api-Request-Id", newVolcengineAgentPlanRequestID())
	headers.Set("X-Control-Require-Usage-Tokens-Return", "*")
	return headers
}

func newVolcengineAgentPlanRequestID() string {
	return uuid.NewString()
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
	req.Header.Set("Content-Type", "application/json")
	if c != nil && c.Request != nil {
		if contentType := strings.TrimSpace(c.GetHeader("Content-Type")); contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if accept := strings.TrimSpace(c.GetHeader("Accept")); accept != "" {
			req.Header.Set("Accept", accept)
		}
	}
	if endpoint == VolcengineAgentPlanImagesGenerations {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		planHeaders := VolcengineAgentPlanHTTPHeaders(endpoint, token)
		for key, values := range planHeaders {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.tlsFPProfileService != nil {
		if tlsProfile := s.tlsFPProfileService.ResolveTLSProfile(account); tlsProfile != nil {
			start := time.Now()
			resp, requestErr := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
			return s.finishVolcengineAgentPlanHTTP(c, resp, requestErr, endpoint, upstreamModel, streamResponse, start)
		}
	}
	start := time.Now()
	resp, requestErr := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	return s.finishVolcengineAgentPlanHTTP(c, resp, requestErr, endpoint, upstreamModel, streamResponse, start)
}

func (s *GatewayService) finishVolcengineAgentPlanHTTP(
	c *gin.Context,
	resp *http.Response,
	requestErr error,
	endpoint VolcengineAgentPlanEndpoint,
	upstreamModel string,
	streamResponse bool,
	start time.Time,
) (*ForwardResult, error) {
	if requestErr != nil {
		return nil, fmt.Errorf("volcengine agent plan upstream request failed: %w", requestErr)
	}
	if resp == nil {
		return nil, fmt.Errorf("volcengine agent plan upstream returned no response")
	}
	defer func() { _ = resp.Body.Close() }()
	providerCode := strings.TrimSpace(resp.Header.Get("X-Api-Status-Code"))
	providerFailed := providerCode != "" && !volcengineAgentPlanSuccessCode(providerCode)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || providerFailed {
		body, err := readVolcengineAgentPlanResponseBody(resp.Body)
		if err != nil {
			return nil, err
		}
		statusCode := resp.StatusCode
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			statusCode = http.StatusBadGateway
		}
		return nil, &VolcengineAgentPlanUpstreamError{
			StatusCode:   statusCode,
			ProviderCode: providerCode,
			Body:         body,
			Headers:      resp.Header.Clone(),
		}
	}
	if streamResponse {
		stats, err := s.streamVolcengineAgentPlanHTTPResponse(c, resp, endpoint)
		if err != nil {
			return nil, err
		}
		if !stats.Successful(endpoint) {
			return nil, fmt.Errorf("volcengine agent plan stream did not complete successfully")
		}
		model := strings.TrimSpace(upstreamModel)
		result := &ForwardResult{
			RequestID:     volcengineAgentPlanResponseRequestID(resp.Header),
			Model:         model,
			UpstreamModel: model,
			RequestCount:  1,
			Stream:        true,
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
		return nil, err
	}

	stats := parseVolcengineAgentPlanResponseStats(endpoint, body)
	if endpoint == VolcengineAgentPlanImagesGenerations && !stats.Successful(endpoint) {
		return nil, &VolcengineAgentPlanUpstreamError{
			StatusCode: http.StatusBadGateway,
			Body:       body,
			Headers:    resp.Header.Clone(),
		}
	}

	if c != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
		contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Data(resp.StatusCode, contentType, body)
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

func (s *GatewayService) streamVolcengineAgentPlanHTTPResponse(c *gin.Context, resp *http.Response, endpoint VolcengineAgentPlanEndpoint) (volcengineAgentPlanResponseStats, error) {
	observer := newVolcengineAgentPlanResponseObserver(endpoint)
	if c == nil {
		_, err := io.Copy(observer, resp.Body)
		return observer.Stats(), err
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	if strings.TrimSpace(c.Writer.Header().Get("Content-Type")) == "" {
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(resp.StatusCode)

	flusher, _ := c.Writer.(http.Flusher)
	writer := io.Writer(c.Writer)
	if flusher != nil {
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
	ImageCount  int
	Completed   bool
	Failed      bool
	ImageEvents int
}

type volcengineAgentPlanResponseObserver struct {
	endpoint VolcengineAgentPlanEndpoint
	pending  []byte
	stats    volcengineAgentPlanResponseStats
}

func newVolcengineAgentPlanResponseObserver(endpoint VolcengineAgentPlanEndpoint) *volcengineAgentPlanResponseObserver {
	return &volcengineAgentPlanResponseObserver{endpoint: endpoint}
}

func (o *volcengineAgentPlanResponseObserver) Write(payload []byte) (int, error) {
	if o == nil {
		return len(payload), nil
	}
	o.pending = append(o.pending, payload...)
	if o.endpoint == VolcengineAgentPlanTTSUnidirectional {
		o.consumeTTSJSON()
		return len(payload), nil
	}
	for {
		index := bytes.IndexByte(o.pending, '\n')
		if index < 0 {
			break
		}
		o.observeLine(o.pending[:index])
		o.pending = o.pending[index+1:]
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
	} else {
		o.observeLine(o.pending)
	}
	o.pending = nil
	return o.stats
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
	for _, key := range []string{"X-Tt-Logid", "X-Api-Connect-Id", "X-Request-Id"} {
		for _, value := range headers.Values(key) {
			allowed.Add(key, value)
		}
	}
	return allowed
}

func volcengineAgentPlanResponseRequestID(headers http.Header) string {
	if requestID := strings.TrimSpace(headers.Get("X-Request-Id")); requestID != "" {
		return requestID
	}
	return strings.TrimSpace(headers.Get("X-Tt-Logid"))
}

func readVolcengineAgentPlanResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, volcengineAgentPlanMaxHTTPResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read Volcengine Agent Plan response: %w", err)
	}
	if len(data) > volcengineAgentPlanMaxHTTPResponseBytes {
		return nil, fmt.Errorf("volcengine agent plan response exceeds %d bytes", volcengineAgentPlanMaxHTTPResponseBytes)
	}
	return data, nil
}

func (s *GatewayService) ProxyVolcengineAgentPlanWebSocket(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	endpoint VolcengineAgentPlanEndpoint,
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
	upstreamConn, upstreamResp, err := dialer.DialContext(ctx, targetURL, VolcengineAgentPlanWebSocketHeaders(endpoint, token))
	if err != nil {
		if upstreamResp != nil {
			defer func() { _ = upstreamResp.Body.Close() }()
			body, _ := io.ReadAll(io.LimitReader(upstreamResp.Body, 2<<20))
			return nil, &VolcengineAgentPlanUpstreamError{
				StatusCode: upstreamResp.StatusCode,
				Body:       body,
				Headers:    upstreamResp.Header.Clone(),
			}
		}
		return nil, fmt.Errorf("connect Volcengine Agent Plan WebSocket: %w", err)
	}
	defer func() { _ = upstreamConn.Close() }()

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
	if requestCount == 0 {
		return nil, fmt.Errorf("volcengine agent plan websocket closed before a successful result: %w", firstErr)
	}
	return &ForwardResult{
		RequestID:     newVolcengineAgentPlanRequestID(),
		Model:         endpoint.BillingModel(),
		UpstreamModel: endpoint.BillingModel(),
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
	if t == nil || !volcengineAgentPlanBusinessSuccess(payload) {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.endpoint == VolcengineAgentPlanTTSBidirection {
		if t.pendingSessions > 0 {
			t.pendingSessions--
			t.requestCount++
		}
		return
	}
	if t.requestCount == 0 {
		t.requestCount = 1
	}
}

func (t *volcengineAgentPlanWSUsageTracker) RequestCount() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.requestCount
}

func volcengineAgentPlanStartSession(payload []byte) bool {
	decoded := strings.ToLower(string(decodeVolcengineAgentPlanPayload(payload)))
	return strings.Contains(decoded, "startsession") || strings.Contains(decoded, "start_session")
}

func volcengineAgentPlanBusinessSuccess(payload []byte) bool {
	decoded := decodeVolcengineAgentPlanPayload(payload)
	if len(bytes.TrimSpace(decoded)) == 0 {
		return false
	}
	lower := strings.ToLower(string(decoded))
	if strings.Contains(lower, "connectionstarted") ||
		strings.Contains(lower, "connection_started") ||
		strings.Contains(lower, "sessionstarted") ||
		strings.Contains(lower, "session_started") {
		return false
	}
	if strings.Contains(lower, `"error"`) ||
		strings.Contains(lower, `"status":"failed"`) ||
		strings.Contains(lower, `"status": "failed"`) {
		return false
	}
	var envelope map[string]any
	if json.Unmarshal(decoded, &envelope) == nil {
		if rawCode, ok := envelope["code"]; ok {
			if !volcengineAgentPlanSuccessValue(rawCode) {
				return false
			}
		}
	}
	return true
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
