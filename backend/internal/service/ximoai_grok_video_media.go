package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const ximoAIGrokVideoDurationSeconds = 10

func (s *OpenAIGatewayService) forwardXimoAIGrokVideoMedia(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	endpoint GrokMediaEndpoint,
	requestID string,
	body []byte,
	contentType string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	method, publicEndpoint, err := ximoAIGrokVideoPublicEndpoint(endpoint, requestID)
	if err != nil {
		return nil, err
	}

	requestInfo := ParseGrokMediaRequest(contentType, body)
	requestModel := requestInfo.Model
	upstreamModel := account.GetMappedModel(requestModel)
	forwardBody := body
	forwardContentType := contentType
	if requestModel != "" && upstreamModel != "" && upstreamModel != requestModel {
		forwardBody, forwardContentType, err = rewriteOpenAIImagesModel(body, contentType, upstreamModel)
		if err != nil {
			return nil, err
		}
	}

	providerReq, err := adaptOpenAIVideoProviderRequest(account, method, publicEndpoint, forwardBody, forwardContentType)
	if err != nil {
		return nil, err
	}
	setOpsUpstreamRequestBody(c, providerReq.Body)

	token, _, err := s.getRequestCredential(ctx, c, account)
	if err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	defer releaseUpstreamCtx()
	upstreamReq, err := s.buildXimoAIGrokVideoRequest(upstreamCtx, c, account, providerReq, token)
	if err != nil {
		return nil, err
	}
	account.ApplyHeaderOverrides(upstreamReq.Header)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, false)
	}
	defer func() { _ = resp.Body.Close() }()

	requestIDHeader := firstNonEmpty(resp.Header.Get("x-request-id"), resp.Header.Get("xai-request-id"))
	if resp.StatusCode >= http.StatusBadRequest {
		return s.handleGrokMediaErrorResponse(ctx, resp, c, account, endpoint, requestIDHeader, requestModel)
	}

	s.updateGrokUsageFromResponse(ctx, account, resp.Header, resp.StatusCode)
	responseBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	responseBody, err = adaptOpenAIVideoProviderResponse(account, publicEndpoint, responseBody)
	if err != nil {
		return nil, err
	}
	writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)

	if endpoint.IsGenerationRequest() {
		requestInfo.DurationSeconds = ximoAIGrokVideoDurationSeconds
	}
	usage := grokMediaUsageFromResponse(endpoint, requestInfo, responseBody)
	return &OpenAIForwardResult{
		RequestID:            requestIDHeader,
		ResponseID:           usage.ResponseID,
		Usage:                usage.Usage,
		Model:                requestModel,
		BillingModel:         requestModel,
		UpstreamModel:        upstreamModel,
		ResponseHeaders:      resp.Header.Clone(),
		Duration:             time.Since(startTime),
		ImageCount:           usage.ImageCount,
		ImageSize:            usage.ImageSize,
		ImageInputSize:       usage.ImageInputSize,
		ImageOutputSizes:     usage.ImageOutputSizes,
		VideoCount:           usage.VideoCount,
		VideoResolution:      usage.VideoResolution,
		VideoDurationSeconds: usage.VideoDurationSeconds,
	}, nil
}

func (s *OpenAIGatewayService) buildXimoAIGrokVideoRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	providerReq providerProtocolRequest,
	token string,
) (*http.Request, error) {
	baseURL, err := s.validateUpstreamBaseURL(account.GetGrokBaseURL())
	if err != nil {
		return nil, err
	}
	targetURL := buildOpenAIVideosURL(baseURL, providerReq.Endpoint)
	if c != nil && c.Request != nil && strings.TrimSpace(c.Request.URL.RawQuery) != "" {
		separator := "?"
		if strings.Contains(targetURL, "?") {
			separator = "&"
		}
		targetURL += separator + c.Request.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(ctx, providerReq.Method, targetURL, bytes.NewReader(providerReq.Body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			if !openaiPassthroughAllowedHeaders[strings.ToLower(key)] {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	if userAgent := account.GetOpenAIUserAgent(); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if contentType := strings.TrimSpace(providerReq.ContentType); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func ximoAIGrokVideoPublicEndpoint(endpoint GrokMediaEndpoint, requestID string) (string, string, error) {
	switch endpoint {
	case GrokMediaEndpointVideosGenerations:
		return http.MethodPost, "/v1/videos/generations", nil
	case GrokMediaEndpointVideosExtensions:
		return http.MethodPost, "/v1/videos/extensions", nil
	case GrokMediaEndpointVideoStatus:
		requestID = strings.TrimSpace(requestID)
		if requestID == "" {
			return "", "", fmt.Errorf("request_id is required")
		}
		return http.MethodGet, "/v1/videos/" + url.PathEscape(requestID), nil
	default:
		return "", "", fmt.Errorf("grok-video does not support media endpoint %s", endpoint)
	}
}

func (s *OpenAIGatewayService) handleGrokMediaAccountUpstreamError(
	ctx context.Context,
	account *Account,
	endpoint GrokMediaEndpoint,
	statusCode int,
	headers http.Header,
	responseBody []byte,
) {
	_ = endpoint
	s.handleGrokAccountUpstreamError(ctx, account, statusCode, headers, responseBody)
}
