package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	openAIVideosEndpoint = "/v1/videos"
	openAIVideosURL      = "https://api.openai.com/v1/videos"
)

var (
	ErrOpenAIVideoJobNotFound       = errors.New("openai video job not found")
	ErrOpenAIVideoCharacterNotFound = errors.New("openai video character not found")
)

type OpenAIVideoJob struct {
	VideoID       string
	Platform      string
	AccountID     int64
	GroupID       *int64
	APIKeyID      *int64
	UserID        *int64
	ChannelID     *int64
	RequestModel  string
	UpstreamModel string
	Status        string
	ResponseJSON  map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OpenAIVideoCharacter struct {
	CharacterID  string
	Platform     string
	AccountID    int64
	GroupID      *int64
	APIKeyID     *int64
	UserID       *int64
	ResponseJSON map[string]any
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OpenAIVideoJobRepository interface {
	GetByVideoID(ctx context.Context, videoID string) (*OpenAIVideoJob, error)
	List(ctx context.Context, filter OpenAIVideoJobListFilter) ([]OpenAIVideoJob, error)
	Upsert(ctx context.Context, job *OpenAIVideoJob) error
	DeleteByVideoID(ctx context.Context, videoID string) error
	GetCharacterByID(ctx context.Context, characterID string) (*OpenAIVideoCharacter, error)
	UpsertCharacter(ctx context.Context, character *OpenAIVideoCharacter) error
}

type OpenAIVideoRequest struct {
	Endpoint       string
	ContentType    string
	Model          string
	Body           []byte
	Multipart      bool
	SourceVideoIDs []string
}

type OpenAIVideoJobMeta struct {
	Platform  string
	GroupID   *int64
	APIKeyID  *int64
	UserID    *int64
	ChannelID *int64
}

type OpenAIVideoJobListFilter struct {
	Platform string
	GroupID  *int64
	UserID   *int64
	After    string
	Limit    int
	Order    string
}

func (s *OpenAIGatewayService) ParseOpenAIVideoRequest(c *gin.Context, body []byte, requireModel bool) (*OpenAIVideoRequest, error) {
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("missing request context")
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	endpoint := normalizeOpenAIVideoEndpointPath(c.Request.URL.Path)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported videos endpoint")
	}
	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	mediaType, _, _ := mime.ParseMediaType(contentType)
	multipartBody := strings.EqualFold(mediaType, "multipart/form-data")

	model, err := extractOpenAIVideoRequestModel(body, contentType)
	if err != nil {
		return nil, err
	}
	if requireModel && strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}

	return &OpenAIVideoRequest{
		Endpoint:       endpoint,
		ContentType:    contentType,
		Model:          strings.TrimSpace(model),
		Body:           body,
		Multipart:      multipartBody,
		SourceVideoIDs: extractOpenAIVideoSourceIDs(body, contentType, endpoint),
	}, nil
}

func normalizeOpenAIVideoEndpointPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "/v1/videos") {
		return path
	}
	if strings.HasPrefix(path, "/videos") {
		return "/v1" + path
	}
	return ""
}

func extractOpenAIVideoRequestModel(body []byte, contentType string) (string, error) {
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

func extractOpenAIVideoSourceIDs(body []byte, contentType string, endpoint string) []string {
	ids := make([]string, 0, 2)
	if strings.HasSuffix(normalizeOpenAIVideoEndpointPath(endpoint), "/remix") {
		ids = append(ids, extractOpenAIVideoIDFromEndpoint(endpoint))
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return uniqueNonEmptyStrings(ids)
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			formName := part.FormName()
			if part.FileName() != "" || (formName != "video" && formName != "video_id" && formName != "source_video_id" && formName != "video[id]") {
				_ = part.Close()
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(part, 4096))
			_ = part.Close()
			if readErr == nil {
				raw := strings.TrimSpace(string(data))
				if gjson.Valid(raw) {
					parsed := gjson.Parse(raw)
					if strings.HasPrefix(raw, "{") {
						ids = append(ids, strings.TrimSpace(parsed.Get("id").String()))
					} else {
						ids = append(ids, strings.TrimSpace(parsed.String()))
					}
				} else {
					ids = append(ids, raw)
				}
			}
		}
		return uniqueNonEmptyStrings(ids)
	}

	if gjson.ValidBytes(body) {
		for _, path := range []string{"video.id", "video_id", "source_video_id", "input_video.id", "input_video_id"} {
			ids = append(ids, strings.TrimSpace(gjson.GetBytes(body, path).String()))
		}
	}
	return uniqueNonEmptyStrings(ids)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (s *OpenAIGatewayService) ForwardOpenAIVideoMutation(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIVideoRequest,
	channelMappedModel string,
	meta OpenAIVideoJobMeta,
) (*OpenAIForwardResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("parsed video request is required")
	}
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("OpenAI videos API only supports API Key accounts")
	}

	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" && requestModel != "" {
		requestModel = mapped
	}
	upstreamModel := requestModel
	if upstreamModel != "" {
		upstreamModel = account.GetMappedModel(upstreamModel)
	}

	forwardBody := body
	forwardContentType := parsed.ContentType
	var err error
	if upstreamModel != "" {
		forwardBody, forwardContentType, err = rewriteOpenAIImagesModel(body, parsed.ContentType, upstreamModel)
		if err != nil {
			return nil, err
		}
	}
	providerReq, err := adaptOpenAIVideoProviderRequest(account, http.MethodPost, parsed.Endpoint, forwardBody, forwardContentType)
	if err != nil {
		return nil, err
	}
	setOpsUpstreamRequestBody(c, providerReq.Body)

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideoRequest(ctx, c, account, providerReq.Method, providerReq.Endpoint, providerReq.Body, providerReq.ContentType, token)
	if err != nil {
		return nil, err
	}

	resp, err := s.doOpenAIVideoUpstream(ctx, c, account, upstreamReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
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
		return nil, s.handleErrorResponsePassthrough(ctx, &copiedResp, c, account, forwardBody)
	}

	responseBody, err := s.writeOpenAIVideoJSONResponse(resp, c)
	if err != nil {
		return nil, err
	}
	responseModel := extractOpenAIVideoModel(responseBody)
	if requestModel == "" {
		requestModel = responseModel
		if requestModel == "" && account.Platform == PlatformOpenAI {
			requestModel = "sora-2"
		}
	}
	if upstreamModel == "" {
		upstreamModel = responseModel
		if upstreamModel == "" && account.Platform == PlatformOpenAI {
			upstreamModel = requestModel
		}
	}
	videoID := extractOpenAIVideoID(responseBody)
	if videoID != "" {
		s.saveOpenAIVideoJob(ctx, videoID, responseBody, account, requestModel, upstreamModel, meta)
	}
	usage, _ := extractOpenAIUsageFromJSONBytes(responseBody)
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		ResponseID:      videoID,
		Usage:           usage,
		Model:           requestModel,
		BillingModel:    requestModel,
		UpstreamModel:   upstreamModel,
		Stream:          false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
		VideoCount:      1,
	}, nil
}

func (s *OpenAIGatewayService) ForwardOpenAIVideoCharacterCreate(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	contentType string,
	meta OpenAIVideoJobMeta,
) (*OpenAIForwardResult, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("OpenAI videos API only supports API Key accounts")
	}

	startTime := time.Now()
	setOpsUpstreamRequestBody(c, body)

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideoRequest(ctx, c, account, http.MethodPost, "/v1/videos/characters", body, contentType, token)
	if err != nil {
		return nil, err
	}

	resp, err := s.doOpenAIVideoUpstream(ctx, c, account, upstreamReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
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
		return nil, s.handleErrorResponsePassthrough(ctx, &copiedResp, c, account, body)
	}

	responseBody, err := s.writeOpenAIVideoJSONResponse(resp, c)
	if err != nil {
		return nil, err
	}
	characterID := extractOpenAIVideoCharacterID(responseBody)
	if characterID != "" {
		s.saveOpenAIVideoCharacter(ctx, characterID, responseBody, account, meta)
	}
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		ResponseID:      characterID,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func (s *OpenAIGatewayService) ForwardOpenAIVideoJSON(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	endpoint string,
) (*OpenAIForwardResult, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("OpenAI videos API only supports API Key accounts")
	}
	endpoint = normalizeOpenAIVideoEndpointPath(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported videos endpoint")
	}
	startTime := time.Now()
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	providerReq, err := adaptOpenAIVideoProviderRequest(account, method, endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideoRequest(ctx, c, account, providerReq.Method, providerReq.Endpoint, providerReq.Body, providerReq.ContentType, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.doOpenAIVideoUpstream(ctx, c, account, upstreamReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, s.handleErrorResponsePassthrough(ctx, resp, c, account, nil)
	}
	body, err := s.writeOpenAIVideoJSONResponse(resp, c)
	if err != nil {
		return nil, err
	}
	videoID := extractOpenAIVideoID(body)
	if videoID == "" {
		videoID = extractOpenAIVideoIDFromEndpoint(endpoint)
	}
	if videoID != "" {
		if strings.EqualFold(method, http.MethodDelete) {
			s.deleteOpenAIVideoJob(ctx, videoID)
		} else {
			s.refreshOpenAIVideoJob(ctx, videoID, body, account)
		}
	}
	usage, _ := extractOpenAIUsageFromJSONBytes(body)
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		ResponseID:      videoID,
		Usage:           usage,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func (s *OpenAIGatewayService) ForwardOpenAIVideoCharacterJSON(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	endpoint string,
) (*OpenAIForwardResult, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("OpenAI videos API only supports API Key accounts")
	}
	endpoint = normalizeOpenAIVideoEndpointPath(endpoint)
	if endpoint == "" || !strings.Contains(endpoint, "/videos/characters/") {
		return nil, fmt.Errorf("unsupported videos character endpoint")
	}
	startTime := time.Now()
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideoRequest(ctx, c, account, method, endpoint, nil, "", token)
	if err != nil {
		return nil, err
	}
	resp, err := s.doOpenAIVideoUpstream(ctx, c, account, upstreamReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, s.handleErrorResponsePassthrough(ctx, resp, c, account, nil)
	}
	body, err := s.writeOpenAIVideoJSONResponse(resp, c)
	if err != nil {
		return nil, err
	}
	characterID := extractOpenAIVideoCharacterID(body)
	if characterID == "" {
		characterID = extractOpenAIVideoCharacterIDFromEndpoint(endpoint)
	}
	if characterID != "" {
		s.refreshOpenAIVideoCharacter(ctx, characterID, body, account)
	}
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		ResponseID:      characterID,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func (s *OpenAIGatewayService) ForwardOpenAIVideoContent(ctx context.Context, c *gin.Context, account *Account, endpoint string) error {
	if account == nil {
		return fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return fmt.Errorf("OpenAI videos API only supports API Key accounts")
	}
	endpoint = normalizeOpenAIVideoEndpointPath(endpoint)
	if endpoint == "" {
		return fmt.Errorf("unsupported videos endpoint")
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return err
	}
	providerReq, err := adaptOpenAIVideoProviderRequest(account, http.MethodGet, endpoint, nil, "")
	if err != nil {
		return err
	}
	upstreamReq, err := s.buildOpenAIVideoRequest(ctx, c, account, providerReq.Method, providerReq.Endpoint, providerReq.Body, providerReq.ContentType, token)
	if err != nil {
		return err
	}
	copyOpenAIVideoContentRequestHeaders(c.Request.Header, upstreamReq.Header)

	resp, err := s.doOpenAIVideoUpstream(ctx, c, account, upstreamReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if account.IsGrokVideo() || account.IsGeminiCompatibleAPIKey() {
		respBody, readErr := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			if videoURL := extractOpenAIVideoURL(respBody); videoURL != "" {
				c.Redirect(http.StatusFound, videoURL)
				return nil
			}
		}
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
		contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, respBody)
		return nil
	}
	writeOpenAIVideoContentResponseHeaders(c.Writer.Header(), resp.Header)
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		return err
	}
	return nil
}

func (s *OpenAIGatewayService) ResolveOpenAIVideoJobAccount(ctx context.Context, videoID string) (*OpenAIVideoJob, *Account, error) {
	if s == nil || s.videoJobRepo == nil {
		return nil, nil, ErrOpenAIVideoJobNotFound
	}
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return nil, nil, ErrOpenAIVideoJobNotFound
	}
	job, err := s.videoJobRepo.GetByVideoID(ctx, videoID)
	if err != nil {
		if !strings.Contains(videoID, "/") {
			if fallbackJob, fallbackErr := s.videoJobRepo.GetByVideoID(ctx, "operations/"+videoID); fallbackErr == nil {
				job = fallbackJob
				err = nil
			}
		}
		if err != nil {
			return nil, nil, err
		}
	}
	account, err := s.accountRepo.GetByID(ctx, job.AccountID)
	if err != nil {
		return nil, nil, err
	}
	return job, account, nil
}

func (s *OpenAIGatewayService) ResolveOpenAIVideoCharacterAccount(ctx context.Context, characterID string) (*OpenAIVideoCharacter, *Account, error) {
	if s == nil || s.videoJobRepo == nil {
		return nil, nil, ErrOpenAIVideoCharacterNotFound
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return nil, nil, ErrOpenAIVideoCharacterNotFound
	}
	character, err := s.videoJobRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, nil, err
	}
	account, err := s.accountRepo.GetByID(ctx, character.AccountID)
	if err != nil {
		return nil, nil, err
	}
	return character, account, nil
}

func (s *OpenAIGatewayService) ListOpenAIVideoJobs(ctx context.Context, filter OpenAIVideoJobListFilter) ([]map[string]any, error) {
	if s == nil || s.videoJobRepo == nil {
		return []map[string]any{}, nil
	}
	filter.Platform = NormalizePlatformSlug(filter.Platform)
	jobs, err := s.videoJobRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(jobs))
	for i := range jobs {
		out = append(out, openAIVideoJobListItem(jobs[i]))
	}
	return out, nil
}

func (s *OpenAIGatewayService) buildOpenAIVideoRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	endpoint string,
	body []byte,
	contentType string,
	token string,
) (*http.Request, error) {
	if account.IsGeminiCompatibleAPIKey() {
		return s.buildGeminiVideoRequest(ctx, c, account, method, endpoint, body, contentType, token)
	}
	targetURL := openAIVideosURL
	baseURL := account.GetOpenAIBaseURL()
	if account.IsGrokVideo() {
		baseURL = account.GetGrokBaseURL()
	}
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = buildOpenAIVideosURL(validatedURL, endpoint)
	}
	if c != nil && c.Request != nil && strings.TrimSpace(c.Request.URL.RawQuery) != "" {
		separator := "?"
		if strings.Contains(targetURL, "?") {
			separator = "&"
		}
		targetURL += separator + c.Request.URL.RawQuery
	}

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
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

func (s *OpenAIGatewayService) buildGeminiVideoRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	endpoint string,
	body []byte,
	contentType string,
	token string,
) (*http.Request, error) {
	baseURL := account.GetGeminiBaseURL(PlatformDefaultBaseURLGemini)
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	targetURL := buildGeminiVideoURL(validatedURL, endpoint)
	if c != nil && c.Request != nil && strings.TrimSpace(c.Request.URL.RawQuery) != "" {
		separator := "?"
		if strings.Contains(targetURL, "?") {
			separator = "&"
		}
		targetURL += separator + c.Request.URL.RawQuery
	}

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", token)
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lowerKey := strings.ToLower(key)
			if lowerKey == "authorization" || lowerKey == "x-goog-api-key" || !openaiPassthroughAllowedHeaders[lowerKey] {
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

func (s *OpenAIGatewayService) doOpenAIVideoUpstream(ctx context.Context, c *gin.Context, account *Account, req *http.Request) (*http.Response, error) {
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

func (s *OpenAIGatewayService) writeOpenAIVideoJSONResponse(resp *http.Response, c *gin.Context) ([]byte, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	c.Data(resp.StatusCode, contentType, body)
	return body, nil
}

func buildOpenAIVideosURL(base string, endpoint string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = openAIVideosEndpoint
	}
	relative := strings.TrimPrefix(endpoint, "/v1")
	if strings.HasSuffix(normalized, endpoint) || strings.HasSuffix(normalized, relative) {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/videos") {
		return normalized + strings.TrimPrefix(endpoint, "/v1/videos")
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + relative
	}
	return normalized + endpoint
}

func buildGeminiVideoURL(base string, endpoint string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "/v1beta"
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	if strings.HasPrefix(endpoint, "/v1beta/") || endpoint == "/v1beta" {
		for _, suffix := range []string{"/v1beta/models", "/v1beta"} {
			if strings.HasSuffix(normalized, suffix) {
				return strings.TrimSuffix(normalized, suffix) + endpoint
			}
		}
	}
	return normalized + endpoint
}

func extractOpenAIVideoID(body []byte) string {
	for _, path := range []string{"id", "video_id", "task_id", "name", "operation.name", "video.id", "data.id", "data.video_id", "data.task_id", "data.name"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func extractOpenAIVideoIDFromEndpoint(endpoint string) string {
	endpoint = strings.Trim(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return ""
	}
	parts := strings.Split(endpoint, "/")
	for i, part := range parts {
		if part == "videos" && i+1 < len(parts) {
			next := strings.TrimSpace(parts[i+1])
			switch next {
			case "", "characters", "edits", "extensions":
				return ""
			case "operations":
				if i+2 >= len(parts) {
					return ""
				}
				value := strings.Join(parts[i+1:], "/")
				value = strings.TrimSuffix(value, "/content")
				if decoded, err := url.PathUnescape(value); err == nil {
					value = decoded
				}
				return strings.TrimSpace(value)
			default:
				if decoded, err := url.PathUnescape(next); err == nil {
					next = decoded
				}
				return strings.TrimSpace(next)
			}
		}
	}
	return ""
}

func extractOpenAIVideoCharacterID(body []byte) string {
	for _, path := range []string{"id", "character_id", "character.id", "data.id"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func extractOpenAIVideoCharacterIDFromEndpoint(endpoint string) string {
	endpoint = strings.Trim(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return ""
	}
	parts := strings.Split(endpoint, "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "videos" && parts[i+1] == "characters" {
			return strings.TrimSpace(parts[i+2])
		}
	}
	return ""
}

func extractOpenAIVideoModel(body []byte) string {
	for _, path := range []string{"model", "video.model", "data.model"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func openAIVideoJobListItem(job OpenAIVideoJob) map[string]any {
	item := map[string]any{}
	for key, value := range job.ResponseJSON {
		item[key] = value
	}
	if strings.TrimSpace(stringValueFromMap(item, "id")) == "" {
		item["id"] = job.VideoID
	}
	if strings.TrimSpace(stringValueFromMap(item, "object")) == "" {
		item["object"] = "video"
	}
	if strings.TrimSpace(stringValueFromMap(item, "model")) == "" {
		if job.RequestModel != "" {
			item["model"] = job.RequestModel
		} else if job.UpstreamModel != "" {
			item["model"] = job.UpstreamModel
		}
	}
	if strings.TrimSpace(stringValueFromMap(item, "status")) == "" && job.Status != "" {
		item["status"] = job.Status
	}
	if _, ok := item["created_at"]; !ok && !job.CreatedAt.IsZero() {
		item["created_at"] = job.CreatedAt.Unix()
	}
	return item
}

func stringValueFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func (s *OpenAIGatewayService) saveOpenAIVideoJob(
	ctx context.Context,
	videoID string,
	body []byte,
	account *Account,
	requestModel string,
	upstreamModel string,
	meta OpenAIVideoJobMeta,
) {
	if s == nil || s.videoJobRepo == nil || account == nil || strings.TrimSpace(videoID) == "" {
		return
	}
	responseJSON := map[string]any{}
	if len(body) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil && parsed != nil {
			responseJSON = parsed
		}
	}
	status := extractOpenAIVideoStatus(body)
	if meta.Platform == "" {
		meta.Platform = account.Platform
	}
	job := &OpenAIVideoJob{
		VideoID:       strings.TrimSpace(videoID),
		Platform:      NormalizePlatformSlug(meta.Platform),
		AccountID:     account.ID,
		GroupID:       meta.GroupID,
		APIKeyID:      meta.APIKeyID,
		UserID:        meta.UserID,
		ChannelID:     meta.ChannelID,
		RequestModel:  strings.TrimSpace(requestModel),
		UpstreamModel: strings.TrimSpace(upstreamModel),
		Status:        status,
		ResponseJSON:  responseJSON,
	}
	if err := s.videoJobRepo.Upsert(ctx, job); err != nil {
		logger.L().With(
			zap.String("component", "service.openai_gateway.videos"),
			zap.String("video_id", videoID),
			zap.Int64("account_id", account.ID),
		).Warn("openai.videos.save_job_failed", zap.Error(err))
	}
}

func (s *OpenAIGatewayService) refreshOpenAIVideoJob(ctx context.Context, videoID string, body []byte, account *Account) {
	if s == nil || s.videoJobRepo == nil || strings.TrimSpace(videoID) == "" || account == nil {
		return
	}
	job, err := s.videoJobRepo.GetByVideoID(ctx, videoID)
	if err != nil || job == nil {
		return
	}
	responseJSON := map[string]any{}
	if len(body) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil && parsed != nil {
			responseJSON = parsed
		}
	}
	status := extractOpenAIVideoStatus(body)
	if status != "" {
		job.Status = status
	}
	if len(responseJSON) > 0 {
		job.ResponseJSON = responseJSON
	}
	job.AccountID = account.ID
	if err := s.videoJobRepo.Upsert(ctx, job); err != nil {
		logger.L().With(
			zap.String("component", "service.openai_gateway.videos"),
			zap.String("video_id", videoID),
			zap.Int64("account_id", account.ID),
		).Warn("openai.videos.refresh_job_failed", zap.Error(err))
	}
}

func (s *OpenAIGatewayService) deleteOpenAIVideoJob(ctx context.Context, videoID string) {
	if s == nil || s.videoJobRepo == nil || strings.TrimSpace(videoID) == "" {
		return
	}
	if err := s.videoJobRepo.DeleteByVideoID(ctx, videoID); err != nil {
		logger.L().With(
			zap.String("component", "service.openai_gateway.videos"),
			zap.String("video_id", videoID),
		).Warn("openai.videos.delete_job_failed", zap.Error(err))
	}
}

func (s *OpenAIGatewayService) saveOpenAIVideoCharacter(
	ctx context.Context,
	characterID string,
	body []byte,
	account *Account,
	meta OpenAIVideoJobMeta,
) {
	if s == nil || s.videoJobRepo == nil || account == nil || strings.TrimSpace(characterID) == "" {
		return
	}
	responseJSON := map[string]any{}
	if len(body) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil && parsed != nil {
			responseJSON = parsed
		}
	}
	if meta.Platform == "" {
		meta.Platform = account.Platform
	}
	character := &OpenAIVideoCharacter{
		CharacterID:  strings.TrimSpace(characterID),
		Platform:     NormalizePlatformSlug(meta.Platform),
		AccountID:    account.ID,
		GroupID:      meta.GroupID,
		APIKeyID:     meta.APIKeyID,
		UserID:       meta.UserID,
		ResponseJSON: responseJSON,
	}
	if err := s.videoJobRepo.UpsertCharacter(ctx, character); err != nil {
		logger.L().With(
			zap.String("component", "service.openai_gateway.videos"),
			zap.String("character_id", characterID),
			zap.Int64("account_id", account.ID),
		).Warn("openai.videos.save_character_failed", zap.Error(err))
	}
}

func (s *OpenAIGatewayService) refreshOpenAIVideoCharacter(ctx context.Context, characterID string, body []byte, account *Account) {
	if s == nil || s.videoJobRepo == nil || strings.TrimSpace(characterID) == "" || account == nil {
		return
	}
	character, err := s.videoJobRepo.GetCharacterByID(ctx, characterID)
	if err != nil || character == nil {
		return
	}
	responseJSON := map[string]any{}
	if len(body) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err == nil && parsed != nil {
			responseJSON = parsed
		}
	}
	if len(responseJSON) > 0 {
		character.ResponseJSON = responseJSON
	}
	character.AccountID = account.ID
	if err := s.videoJobRepo.UpsertCharacter(ctx, character); err != nil {
		logger.L().With(
			zap.String("component", "service.openai_gateway.videos"),
			zap.String("character_id", characterID),
			zap.Int64("account_id", account.ID),
		).Warn("openai.videos.refresh_character_failed", zap.Error(err))
	}
}

func copyOpenAIVideoContentRequestHeaders(src http.Header, dst http.Header) {
	for _, key := range []string{"Range", "If-Range", "If-None-Match", "If-Modified-Since"} {
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

func writeOpenAIVideoContentResponseHeaders(dst http.Header, src http.Header) {
	for _, key := range []string{
		"Content-Type",
		"Content-Length",
		"Content-Range",
		"Accept-Ranges",
		"ETag",
		"Last-Modified",
		"Cache-Control",
		"Content-Disposition",
	} {
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
