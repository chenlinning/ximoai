package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

func normalizeOpenAIResponsesPassthroughEndpointPath(path string) string {
	path = strings.TrimSpace(path)
	switch {
	case strings.HasPrefix(path, "/v1/responses/"):
		return path
	case strings.HasPrefix(path, "/responses/"):
		return "/v1" + path
	default:
		return ""
	}
}

func ExtractOpenAIResponsesPassthroughID(path string) string {
	endpoint := normalizeOpenAIResponsesPassthroughEndpointPath(path)
	if endpoint == "" {
		return ""
	}
	subpath := strings.TrimPrefix(endpoint, "/v1/responses/")
	if subpath == endpoint {
		return ""
	}
	if idx := strings.Index(subpath, "/"); idx >= 0 {
		subpath = subpath[:idx]
	}
	return strings.TrimSpace(subpath)
}

func (s *OpenAIGatewayService) ForwardOpenAIResponsesPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	body []byte,
) error {
	if account == nil {
		return fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return fmt.Errorf("OpenAI Responses passthrough only supports API Key accounts")
	}
	if c == nil || c.Request == nil {
		return fmt.Errorf("request context is required")
	}

	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return err
	}
	endpoint := normalizeOpenAIResponsesPassthroughEndpointPath(c.Request.URL.Path)
	if endpoint == "" {
		return fmt.Errorf("unsupported responses passthrough endpoint")
	}

	targetURL := buildOpenAIEndpointURL(validatedURL, endpoint)
	if strings.TrimSpace(c.Request.URL.RawQuery) != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "Bearer "+token)
	if c.Request != nil {
		for key, values := range c.Request.Header {
			lower := strings.ToLower(strings.TrimSpace(key))
			if !openaiPassthroughAllowedHeaders[lower] {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Set("authorization", "Bearer "+token)
	if customUA := strings.TrimSpace(account.GetOpenAIUserAgent()); customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if strings.TrimSpace(method) == http.MethodGet || strings.TrimSpace(method) == http.MethodDelete {
		req.Body = nil
		req.ContentLength = 0
	}
	if len(body) == 0 {
		req.Header.Del("content-type")
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			UpstreamURL:        safeUpstreamURL(req.URL.String()),
			Kind:               "request_error",
			Message:            safeErr,
		})
		return fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		copiedResp := *resp
		copiedResp.Body = io.NopCloser(bytes.NewReader(respBody))
		return s.handleErrorResponsePassthrough(ctx, &copiedResp, c, account, body)
	}

	bodyBytes, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return err
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, bodyBytes)
	return nil
}
