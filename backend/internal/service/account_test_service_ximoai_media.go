package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	defaultOpenAIAudioTestInput       = "hi"
	defaultOpenAIVideoTestPrompt      = "A tiny test video of a sunrise over mountains."
	defaultOpenAIVideoSeconds         = 4
	defaultOpenAIVideoSize            = "720x1280"
	accountTestMediaPreviewLimitBytes = 40 * 1024 * 1024
	openAIVideoTestPollAttempts       = 8
	openAIVideoTestPollInterval       = 2 * time.Second
)

var accountTestSleep = func(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isOpenAIAudioModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(model, "audio") ||
		strings.Contains(model, "realtime") ||
		strings.Contains(model, "tts") ||
		strings.HasPrefix(model, "whisper")
}

func isOpenAIChatAudioModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if !strings.Contains(model, "audio") {
		return false
	}
	return !strings.Contains(model, "tts") &&
		!strings.Contains(model, "speech") &&
		!strings.Contains(model, "transcrib") &&
		!strings.Contains(model, "transcript") &&
		!strings.Contains(model, "translat") &&
		!strings.Contains(model, "whisper") &&
		!strings.Contains(model, "stt")
}

func isOpenAIVideoModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(model, "video") ||
		strings.HasPrefix(model, "sora-") ||
		strings.HasPrefix(model, "veo") ||
		strings.Contains(model, "t2v") ||
		strings.Contains(model, "i2v") ||
		strings.Contains(model, "r2v")
}

func (s *AccountTestService) testOpenAIChatAudioAPIKey(c *gin.Context, ctx context.Context, account *Account, modelID string, input string) error {
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}

	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
	}
	apiURL := buildOpenAIChatCompletionsURL(normalizedBaseURL)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})
	s.sendEvent(c, TestEvent{Type: "status", Text: "正在通过 /v1/chat/completions 测试音频对话"})

	audioInput := strings.TrimSpace(input)
	if audioInput == "" {
		audioInput = defaultOpenAIAudioTestInput
	}
	payload := map[string]any{
		"model":      modelID,
		"modalities": []string{"text", "audio"},
		"audio": map[string]any{
			"voice":  "alloy",
			"format": "wav",
		},
		"messages": []map[string]any{{
			"role":    "user",
			"content": audioInput,
		}},
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create audio chat request")
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Audio chat API (/v1/chat/completions) request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, accountTestMediaPreviewLimitBytes+1))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to read audio chat response: %s", err.Error()))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Audio chat API (/v1/chat/completions) returned %d: %s", resp.StatusCode, string(body)))
	}
	if len(body) > accountTestMediaPreviewLimitBytes {
		return s.sendErrorAndEnd(c, "Audio chat response is too large to preview")
	}

	audioB64 := strings.TrimSpace(gjson.GetBytes(body, "choices.0.message.audio.data").String())
	if audioB64 == "" {
		return s.sendErrorAndEnd(c, "No audio data returned from chat completions response")
	}
	audioBytes, err := base64.StdEncoding.DecodeString(audioB64)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to decode audio data: %s", err.Error()))
	}
	if len(audioBytes) > accountTestMediaPreviewLimitBytes {
		return s.sendErrorAndEnd(c, "Decoded audio response is too large to preview")
	}
	if transcript := strings.TrimSpace(gjson.GetBytes(body, "choices.0.message.audio.transcript").String()); transcript != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: transcript})
	}
	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Audio chat test returned %d bytes", len(audioBytes))})
	s.sendEvent(c, TestEvent{
		Type:     "audio",
		AudioURL: "data:audio/wav;base64," + audioB64,
		MimeType: "audio/wav",
	})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

// testOpenAIAudioAPIKey tests OpenAI-compatible speech generation using an API Key account.
func (s *AccountTestService) testOpenAIAudioAPIKey(c *gin.Context, ctx context.Context, account *Account, modelID string, input string) error {
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}

	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})

	audioInput := strings.TrimSpace(input)
	if audioInput == "" {
		audioInput = defaultOpenAIAudioTestInput
	}

	payload := map[string]any{
		"model": modelID,
		"input": audioInput,
		"voice": "alloy",
	}
	if account.IsKlingAudio() {
		payload["voice"] = "genshin_vindi2"
		payload["voice_language"] = "zh"
		payload["voice_speed"] = 1.0
	}
	payloadBytes, _ := json.Marshal(payload)
	providerReq, err := adaptOpenAIAudioProviderRequest(account, openAIAudioSpeechEndpoint, payloadBytes, "application/json")
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}
	apiURL := buildOpenAIEndpointURL(normalizedBaseURL, providerReq.Endpoint)

	req, err := http.NewRequestWithContext(ctx, providerReq.Method, apiURL, bytes.NewReader(providerReq.Body))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	if strings.TrimSpace(providerReq.ContentType) != "" {
		req.Header.Set("Content-Type", providerReq.ContentType)
	}
	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("Authorization", "Bearer "+authToken)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, accountTestMediaPreviewLimitBytes+1))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to read response: %s", err.Error()))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return s.sendErrorAndEnd(c, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}
	if len(body) > accountTestMediaPreviewLimitBytes {
		return s.sendErrorAndEnd(c, "Audio response is too large to preview")
	}
	if account.IsKlingAudio() && gjson.ValidBytes(body) {
		if audioURL := extractOpenAIAudioURL(body); audioURL != "" {
			s.sendEvent(c, TestEvent{
				Type:     "audio",
				AudioURL: audioURL,
				MimeType: "audio/mpeg",
				Data:     json.RawMessage(body),
			})
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}
		s.sendEvent(c, TestEvent{Type: "content", Text: string(body)})
		s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
		return nil
	}

	mimeType := normalizeMediaContentType(resp.Header.Get("Content-Type"), "audio/mpeg")
	s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Audio test returned %d bytes", len(body))})
	s.sendEvent(c, TestEvent{
		Type:     "audio",
		AudioURL: mediaDataURL(mimeType, body),
		MimeType: mimeType,
	})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

// testOpenAIVideoAPIKey tests OpenAI-compatible video creation using an API Key account.
func (s *AccountTestService) testOpenAIVideoAPIKey(c *gin.Context, ctx context.Context, account *Account, modelID string, opts AccountConnectionTestOptions) error {
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}

	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})

	videoPrompt := strings.TrimSpace(opts.Prompt)
	if videoPrompt == "" {
		videoPrompt = defaultOpenAIVideoTestPrompt
	}
	videoSeconds := opts.Seconds
	if videoSeconds <= 0 {
		videoSeconds = defaultOpenAIVideoSeconds
	}
	videoSize := strings.TrimSpace(opts.Size)
	if videoSize == "" {
		videoSize = defaultOpenAIVideoSize
	}
	if account.IsGrokVideo() {
		videoSize = defaultGrokVideoSize
	}

	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	if err := writer.WriteField("model", modelID); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create video request: %s", err.Error()))
	}
	if err := writer.WriteField("prompt", videoPrompt); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create video request: %s", err.Error()))
	}
	if err := writer.WriteField("seconds", fmt.Sprintf("%d", videoSeconds)); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create video request: %s", err.Error()))
	}
	if err := writer.WriteField("size", videoSize); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create video request: %s", err.Error()))
	}
	if account.IsGrokVideo() {
		if err := writer.WriteField("aspect_ratio", defaultGrokVideoAspectRatio); err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create video request: %s", err.Error()))
		}
	}
	if err := writer.Close(); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create video request: %s", err.Error()))
	}
	providerReq, err := adaptOpenAIVideoProviderRequest(account, http.MethodPost, openAIVideosEndpoint, requestBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}
	apiURL := buildOpenAIVideosURL(normalizedBaseURL, providerReq.Endpoint)

	req, err := http.NewRequestWithContext(ctx, providerReq.Method, apiURL, bytes.NewReader(providerReq.Body))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	if strings.TrimSpace(providerReq.ContentType) != "" {
		req.Header.Set("Content-Type", providerReq.ContentType)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, accountTestMediaPreviewLimitBytes+1))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to read response: %s", err.Error()))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return s.sendErrorAndEnd(c, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}
	if len(body) > accountTestMediaPreviewLimitBytes {
		return s.sendErrorAndEnd(c, "Video response is too large to preview")
	}

	if isMediaContentType(resp.Header.Get("Content-Type"), "video/") {
		mimeType := normalizeMediaContentType(resp.Header.Get("Content-Type"), "video/mp4")
		s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Video test returned %d bytes", len(body))})
		s.sendEvent(c, TestEvent{
			Type:     "video",
			VideoURL: mediaDataURL(mimeType, body),
			MimeType: mimeType,
		})
		s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
		return nil
	}

	message := "Video test request accepted"
	videoID := extractOpenAIVideoID(body)
	if videoID != "" {
		message = fmt.Sprintf("Video test created: %s", videoID)
	}
	s.sendEvent(c, TestEvent{Type: "content", Text: message})
	if videoURL := extractOpenAIVideoURL(body); videoURL != "" {
		s.sendEvent(c, TestEvent{
			Type:     "video",
			VideoURL: videoURL,
			MimeType: "video/mp4",
			Data:     json.RawMessage(body),
		})
		s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
		return nil
	}
	if videoID != "" {
		return s.pollOpenAIVideoTestResult(c, ctx, account, normalizedBaseURL, videoID)
	}
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

func (s *AccountTestService) pollOpenAIVideoTestResult(c *gin.Context, ctx context.Context, account *Account, baseURL, videoID string) error {
	lastStatus := ""
	for attempt := 0; attempt < openAIVideoTestPollAttempts; attempt++ {
		if attempt > 0 {
			if err := accountTestSleep(ctx, openAIVideoTestPollInterval); err != nil {
				return s.sendErrorAndEnd(c, err.Error())
			}
		}

		statusEndpoint := fmt.Sprintf("/v1/videos/%s", url.PathEscape(videoID))
		if account.IsGrokVideo() {
			statusEndpoint = "/v1/video/query?id=" + url.QueryEscape(videoID)
		}
		statusCode, headers, body, err := s.doOpenAIVideoTestGet(ctx, account, baseURL, statusEndpoint, "application/json")
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to fetch video status: %s", err.Error()))
		}
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Video status fetch returned %d: %s", statusCode, string(body))})
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}

		if videoURL := extractOpenAIVideoURL(body); videoURL != "" {
			s.sendEvent(c, TestEvent{
				Type:     "video",
				VideoURL: videoURL,
				MimeType: "video/mp4",
				Data:     json.RawMessage(body),
			})
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		}

		status := extractOpenAIVideoStatus(body)
		if status != "" && status != lastStatus {
			s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Video status: %s", status)})
			lastStatus = status
		}
		switch strings.ToLower(status) {
		case "completed", "succeeded", "success":
			contentStatus, contentHeaders, contentBody, err := s.doOpenAIVideoTestGet(ctx, account, baseURL, fmt.Sprintf("/v1/videos/%s/content", url.PathEscape(videoID)), "video/*")
			if err != nil {
				return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to fetch video content: %s", err.Error()))
			}
			if contentStatus >= http.StatusOK && contentStatus < http.StatusMultipleChoices {
				if len(contentBody) > 0 && isMediaContentType(contentHeaders.Get("Content-Type"), "video/") {
					mimeType := normalizeMediaContentType(contentHeaders.Get("Content-Type"), "video/mp4")
					s.sendEvent(c, TestEvent{
						Type:     "video",
						VideoURL: mediaDataURL(mimeType, contentBody),
						MimeType: mimeType,
					})
					s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
					return nil
				}
				if videoURL := extractOpenAIVideoURL(contentBody); videoURL != "" {
					s.sendEvent(c, TestEvent{
						Type:     "video",
						VideoURL: videoURL,
						MimeType: "video/mp4",
					})
					s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
					return nil
				}
			}
			s.sendEvent(c, TestEvent{Type: "content", Text: fmt.Sprintf("Video completed: %s", videoID)})
			s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
			return nil
		case "failed", "error", "cancelled", "canceled":
			errorMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
			if errorMsg == "" {
				errorMsg = fmt.Sprintf("Video test failed with status: %s", status)
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}

		_ = headers
	}

	return s.sendErrorAndEnd(c, fmt.Sprintf("Video test still processing: %s", videoID))
}

func (s *AccountTestService) doOpenAIVideoTestGet(ctx context.Context, account *Account, baseURL, endpoint, accept string) (int, http.Header, []byte, error) {
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return 0, nil, nil, fmt.Errorf("no API key available")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildOpenAIVideosURL(baseURL, endpoint), nil)
	if err != nil {
		return 0, nil, nil, err
	}
	if strings.TrimSpace(accept) == "" {
		accept = "application/json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Authorization", "Bearer "+authToken)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, accountTestMediaPreviewLimitBytes+1))
	if err != nil {
		return resp.StatusCode, resp.Header.Clone(), nil, err
	}
	if len(body) > accountTestMediaPreviewLimitBytes {
		return resp.StatusCode, resp.Header.Clone(), nil, fmt.Errorf("video response is too large to preview")
	}
	return resp.StatusCode, resp.Header.Clone(), body, nil
}

func mediaDataURL(mimeType string, body []byte) string {
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(body))
}

func normalizeMediaContentType(contentType, fallback string) string {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	if contentType == "" {
		return fallback
	}
	return strings.ToLower(contentType)
}

func isMediaContentType(contentType, prefix string) bool {
	return strings.HasPrefix(normalizeMediaContentType(contentType, ""), strings.ToLower(prefix))
}

func extractOpenAIVideoStatus(body []byte) string {
	for _, path := range []string{"status", "task_status", "video.status", "data.status"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	done := gjson.GetBytes(body, "done")
	if done.Exists() {
		if done.Bool() {
			return "succeeded"
		}
		return "running"
	}
	return ""
}

func extractOpenAIVideoURL(body []byte) string {
	for _, path := range []string{
		"url",
		"video_url",
		"content_url",
		"download_url",
		"output_url",
		"metadata.url",
		"video.url",
		"video.download_url",
		"data.url",
		"data.video_url",
		"data.content_url",
		"data.download_url",
		"data.metadata.url",
		"data.0.url",
		"output.url",
		"output.0.url",
		"result.url",
		"response.generatedVideos.0.video.uri",
		"generatedVideos.0.video.uri",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); isPreviewableMediaURL(value) {
			return value
		}
	}
	return ""
}

func extractOpenAIAudioURL(body []byte) string {
	for _, path := range []string{
		"url",
		"audio_url",
		"data.url",
		"data.audio_url",
		"data.task_result.audios.0.url",
		"task_result.audios.0.url",
		"result.url",
		"output.url",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); isPreviewableMediaURL(value) {
			return value
		}
	}
	return ""
}

func isPreviewableMediaURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "data":
		return true
	default:
		return false
	}
}
