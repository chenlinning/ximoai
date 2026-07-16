package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	openAIAudioSpeechEndpoint         = "/v1/audio/speech"
	openAIAudioTranscriptionsEndpoint = "/v1/audio/transcriptions"
	openAIAudioTranslationsEndpoint   = "/v1/audio/translations"
)

type OpenAIAudioRequest struct {
	Endpoint    string
	ContentType string
	Model       string
	Body        []byte
	Multipart   bool
}

func (s *OpenAIGatewayService) ParseOpenAIAudioRequest(c *gin.Context, body []byte) (*OpenAIAudioRequest, error) {
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("missing request context")
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	endpoint := normalizeOpenAIAudioEndpointPath(c.Request.URL.Path)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported audio endpoint")
	}
	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	mediaType, _, _ := mime.ParseMediaType(contentType)
	multipartBody := strings.EqualFold(mediaType, "multipart/form-data")

	model, err := extractOpenAIAudioRequestModel(body, contentType)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}

	return &OpenAIAudioRequest{
		Endpoint:    endpoint,
		ContentType: contentType,
		Model:       strings.TrimSpace(model),
		Body:        body,
		Multipart:   multipartBody,
	}, nil
}

func normalizeOpenAIAudioEndpointPath(path string) string {
	path = strings.TrimSpace(path)
	switch {
	case strings.HasPrefix(path, openAIAudioSpeechEndpoint):
		return openAIAudioSpeechEndpoint
	case strings.HasPrefix(path, openAIAudioTranscriptionsEndpoint):
		return openAIAudioTranscriptionsEndpoint
	case strings.HasPrefix(path, openAIAudioTranslationsEndpoint):
		return openAIAudioTranslationsEndpoint
	case strings.HasPrefix(path, "/audio/speech"):
		return openAIAudioSpeechEndpoint
	case strings.HasPrefix(path, "/audio/transcriptions"):
		return openAIAudioTranscriptionsEndpoint
	case strings.HasPrefix(path, "/audio/translations"):
		return openAIAudioTranslationsEndpoint
	default:
		return ""
	}
}

func extractOpenAIAudioRequestModel(body []byte, contentType string) (string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return "", fmt.Errorf("multipart boundary is required")
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("read multipart body: %w", err)
			}
			if part.FormName() != "model" || part.FileName() != "" {
				_ = part.Close()
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(part, 4096))
			_ = part.Close()
			if readErr != nil {
				return "", fmt.Errorf("read multipart model: %w", readErr)
			}
			return strings.TrimSpace(string(data)), nil
		}
		return "", nil
	}

	if !gjson.ValidBytes(body) {
		return "", fmt.Errorf("failed to parse request body")
	}
	model := gjson.GetBytes(body, "model")
	if model.Exists() && model.Type != gjson.String {
		return "", fmt.Errorf("model must be a string")
	}
	return strings.TrimSpace(model.String()), nil
}

func (s *OpenAIGatewayService) ForwardAudio(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIAudioRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("OpenAI audio API only supports API Key accounts")
	}
	if parsed == nil {
		return nil, fmt.Errorf("parsed audio request is required")
	}

	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	upstreamModel := account.GetMappedModel(requestModel)
	forwardBody, forwardContentType, err := rewriteOpenAIAudioModel(body, parsed.ContentType, upstreamModel)
	if err != nil {
		return nil, err
	}
	providerReq, err := adaptOpenAIAudioProviderRequest(account, parsed.Endpoint, forwardBody, forwardContentType)
	if err != nil {
		return nil, err
	}
	setOpsUpstreamRequestBody(c, providerReq.Body)

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIAudioRequest(ctx, c, account, providerReq.Method, providerReq.Endpoint, providerReq.Body, providerReq.ContentType, token)
	if err != nil {
		return nil, err
	}

	resp, err := s.doOpenAIAudioUpstream(ctx, c, account, upstreamReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if isKlingAudioInvalidVoiceResponse(account, upstreamMsg, respBody) {
			if c != nil && !IsResponseCommitted(c) {
				c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
					"message": "Voice id not found; use a Kling voice_id, not an OpenAI voice name",
					"type":    "invalid_request_error",
					"param":   "voice_id",
				}})
			}
			return nil, fmt.Errorf("kling audio invalid voice_id: %s", upstreamMsg)
		}
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			if s.rateLimitService != nil {
				_ = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				ResponseHeaders:        resp.Header.Clone(),
				RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		copiedResp := *resp
		copiedResp.Body = io.NopCloser(bytes.NewReader(respBody))
		return nil, s.handleErrorResponsePassthrough(ctx, &copiedResp, c, account, forwardBody, respBody)
	}

	usage, responseID, responseModel, err := s.writeOpenAIAudioResponse(resp, c)
	if err != nil {
		return nil, err
	}
	if requestModel == "" {
		requestModel = responseModel
	}
	if upstreamModel == "" {
		upstreamModel = responseModel
	}
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		ResponseID:      responseID,
		Usage:           usage,
		Model:           requestModel,
		BillingModel:    requestModel,
		UpstreamModel:   upstreamModel,
		Stream:          false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func isKlingAudioInvalidVoiceResponse(account *Account, upstreamMsg string, respBody []byte) bool {
	if account == nil || NormalizePlatformSlug(account.Platform) != PlatformKlingAudio {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(upstreamMsg))
	if message == "" && len(respBody) > 0 {
		message = strings.ToLower(string(respBody))
	}
	return strings.Contains(message, "voice id not found")
}

func (s *OpenAIGatewayService) buildOpenAIAudioRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	endpoint string,
	body []byte,
	contentType string,
	token string,
) (*http.Request, error) {
	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	targetURL := buildOpenAIEndpointURL(validatedURL, endpoint)
	if c != nil && c.Request != nil && strings.TrimSpace(c.Request.URL.RawQuery) != "" {
		separator := "?"
		if strings.Contains(targetURL, "?") {
			separator = "&"
		}
		targetURL += separator + c.Request.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
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
	customUA := account.GetOpenAIUserAgent()
	if customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func (s *OpenAIGatewayService) doOpenAIAudioUpstream(ctx context.Context, c *gin.Context, account *Account, req *http.Request) (*http.Response, error) {
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
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	return resp, nil
}

func (s *OpenAIGatewayService) writeOpenAIAudioResponse(resp *http.Response, c *gin.Context) (OpenAIUsage, string, string, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return OpenAIUsage{}, "", "", err
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Data(resp.StatusCode, contentType, body)

	if gjson.ValidBytes(body) {
		usage, _ := extractOpenAIUsageFromJSONBytes(body)
		return usage, strings.TrimSpace(gjson.GetBytes(body, "id").String()), strings.TrimSpace(gjson.GetBytes(body, "model").String()), nil
	}
	return OpenAIUsage{}, "", "", nil
}

func rewriteOpenAIAudioModel(body []byte, contentType string, model string) ([]byte, string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return body, contentType, nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		return rewriteOpenAIImagesMultipartModel(body, contentType, model)
	}
	rewritten, err := sjson.SetBytes(body, "model", model)
	if err != nil {
		return nil, "", fmt.Errorf("rewrite audio request model: %w", err)
	}
	return rewritten, contentType, nil
}
