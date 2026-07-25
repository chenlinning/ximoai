package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	case VolcengineAgentPlanTTSUnidirectional,
		VolcengineAgentPlanTTSUnidirectionalStream,
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

func newVolcengineAgentPlanRequestID() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

type VolcengineAgentPlanUpstreamError struct {
	StatusCode   int
	ProviderCode string
	Body         []byte
	Headers      http.Header
}

func (e *VolcengineAgentPlanUpstreamError) Error() string {
	if e == nil {
		return "Volcengine Agent Plan upstream error"
	}
	return fmt.Sprintf("Volcengine Agent Plan upstream returned HTTP %d", e.StatusCode)
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
		return nil, fmt.Errorf("HTTP upstream is not configured")
	}
	if account == nil || account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("Volcengine Agent Plan requires an API Key account")
	}
	if endpoint.IsWebSocket() {
		return nil, fmt.Errorf("WebSocket endpoint cannot be forwarded over HTTP")
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
	if endpoint == VolcengineAgentPlanImagesGenerations && strings.TrimSpace(upstreamModel) != "" {
		forwardBody, err = sjson.SetBytes(body, "model", upstreamModel)
		if err != nil {
			return nil, fmt.Errorf("rewrite Volcengine image model: %w", err)
		}
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
		planHeaders := VolcengineAgentPlanWebSocketHeaders(VolcengineAgentPlanTTSUnidirectional, token)
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
			return s.finishVolcengineAgentPlanHTTP(c, resp, requestErr, endpoint, upstreamModel, start)
		}
	}
	start := time.Now()
	resp, requestErr := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	return s.finishVolcengineAgentPlanHTTP(c, resp, requestErr, endpoint, upstreamModel, start)
}

func (s *GatewayService) finishVolcengineAgentPlanHTTP(
	c *gin.Context,
	resp *http.Response,
	requestErr error,
	endpoint VolcengineAgentPlanEndpoint,
	upstreamModel string,
	start time.Time,
) (*ForwardResult, error) {
	if requestErr != nil {
		return nil, fmt.Errorf("Volcengine Agent Plan upstream request failed: %w", requestErr)
	}
	if resp == nil {
		return nil, fmt.Errorf("Volcengine Agent Plan upstream returned no response")
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readVolcengineAgentPlanResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}
	providerCode := strings.TrimSpace(resp.Header.Get("X-Api-Status-Code"))
	providerFailed := providerCode != "" && !volcengineAgentPlanSuccessCode(providerCode)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || providerFailed {
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
		RequestID:     strings.TrimSpace(resp.Header.Get("X-Request-Id")),
		Model:         model,
		UpstreamModel: model,
		RequestCount:  1,
		Duration:      time.Since(start),
	}
	if endpoint == VolcengineAgentPlanImagesGenerations {
		result.ImageCount = 1
	}
	return result, nil
}

func readVolcengineAgentPlanResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, volcengineAgentPlanMaxHTTPResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read Volcengine Agent Plan response: %w", err)
	}
	if len(data) > volcengineAgentPlanMaxHTTPResponseBytes {
		return nil, fmt.Errorf("Volcengine Agent Plan response exceeds %d bytes", volcengineAgentPlanMaxHTTPResponseBytes)
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
		return nil, fmt.Errorf("Volcengine Agent Plan requires an API Key account")
	}
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("missing WebSocket request context")
	}
	if !endpoint.IsWebSocket() {
		return nil, fmt.Errorf("HTTP endpoint cannot be forwarded over WebSocket")
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

	clientConn, err := (&websocket.Upgrader{
		ReadBufferSize:  32 << 10,
		WriteBufferSize: 32 << 10,
	}).Upgrade(c.Writer, c.Request, nil)
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
		return nil, fmt.Errorf("Volcengine Agent Plan WebSocket closed before a successful result: %w", firstErr)
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
