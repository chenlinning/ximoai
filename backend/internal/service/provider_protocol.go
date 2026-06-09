package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	klingAudioTTSModel          = "kling-audio-tts"
	klingCustomVoicesModel      = "kling-custom-voices"
	klingPresetVoicesModel      = "kling-presets-voices"
	defaultGrokVideoAspectRatio = "3:2"
	defaultGrokVideoSize        = "720P"
	defaultKlingVoiceLanguage   = "zh"
)

type providerProtocolRequest struct {
	Method      string
	Endpoint    string
	Body        []byte
	ContentType string
}

func adaptOpenAIVideoProviderRequest(account *Account, method string, endpoint string, body []byte, contentType string) (providerProtocolRequest, error) {
	out := providerProtocolRequest{Method: method, Endpoint: endpoint, Body: body, ContentType: contentType}
	if account == nil || NormalizePlatformSlug(account.Platform) != PlatformGrok {
		return out, nil
	}

	videoID := extractOpenAIVideoIDFromEndpoint(endpoint)
	if method == http.MethodGet && videoID != "" {
		out.Endpoint = "/v1/video/query?id=" + url.QueryEscape(videoID)
		return out, nil
	}

	payload, err := parseProviderProtocolPayload(body, contentType)
	if err != nil {
		return out, err
	}
	if endpoint == "/v1/videos/extensions" {
		return buildGrokVideoExtendRequest(payload)
	}
	return buildGrokVideoCreateRequest(payload)
}

func adaptOpenAIAudioProviderRequest(account *Account, endpoint string, body []byte, contentType string) (providerProtocolRequest, error) {
	out := providerProtocolRequest{Method: http.MethodPost, Endpoint: endpoint, Body: body, ContentType: contentType}
	if account == nil || NormalizePlatformSlug(account.Platform) != PlatformKlingAudio {
		return out, nil
	}
	if endpoint != openAIAudioSpeechEndpoint {
		return out, nil
	}

	payload, err := parseProviderProtocolPayload(body, contentType)
	if err != nil {
		return out, err
	}
	model := strings.ToLower(strings.TrimSpace(providerString(payload, "model")))
	switch model {
	case klingAudioTTSModel:
		return buildKlingAudioTTSRequest(payload)
	case klingCustomVoicesModel:
		return buildKlingCustomVoicesRequest(payload)
	case klingPresetVoicesModel:
		return buildKlingPresetVoicesRequest(payload)
	default:
		return out, nil
	}
}

func buildGrokVideoCreateRequest(payload map[string]any) (providerProtocolRequest, error) {
	model := strings.TrimSpace(providerString(payload, "model"))
	prompt := firstProviderString(payload, "prompt", "input")
	if model == "" {
		return providerProtocolRequest{}, fmt.Errorf("model is required")
	}
	if prompt == "" {
		return providerProtocolRequest{}, fmt.Errorf("prompt is required")
	}
	body := map[string]any{
		"model":        model,
		"prompt":       prompt,
		"aspect_ratio": providerStringDefault(payload, defaultGrokVideoAspectRatio, "aspect_ratio"),
		"size":         providerStringDefault(payload, defaultGrokVideoSize, "size"),
	}
	if images := providerStringSlice(payload, "images", "image_urls", "reference_images"); len(images) > 0 {
		body["images"] = images
	}
	return jsonProviderRequest(http.MethodPost, "/v1/video/create", body)
}

func buildGrokVideoExtendRequest(payload map[string]any) (providerProtocolRequest, error) {
	model := strings.TrimSpace(providerString(payload, "model"))
	prompt := firstProviderString(payload, "prompt", "input")
	taskID := firstProviderString(payload, "task_id", "video_id", "id", "video.id")
	if taskID == "" {
		return providerProtocolRequest{}, fmt.Errorf("task_id is required")
	}
	body := map[string]any{
		"task_id": taskID,
	}
	if model != "" {
		body["model"] = model
	}
	if prompt != "" {
		body["prompt"] = prompt
	}
	if value, ok := providerValue(payload, "start_time"); ok {
		body["start_time"] = value
	}
	if value := providerString(payload, "aspect_ratio"); value != "" {
		body["aspect_ratio"] = value
	}
	if value := providerString(payload, "size"); value != "" {
		body["size"] = value
	}
	return jsonProviderRequest(http.MethodPost, "/v1/video/extend", body)
}

func buildKlingAudioTTSRequest(payload map[string]any) (providerProtocolRequest, error) {
	text := firstProviderString(payload, "text", "input", "prompt")
	voiceID := firstProviderString(payload, "voice_id", "voice")
	if text == "" {
		return providerProtocolRequest{}, fmt.Errorf("text is required")
	}
	if voiceID == "" {
		return providerProtocolRequest{}, fmt.Errorf("voice_id is required")
	}
	body := map[string]any{
		"text":           text,
		"voice_id":       voiceID,
		"voice_language": providerStringDefault(payload, defaultKlingVoiceLanguage, "voice_language", "language"),
	}
	if value, ok := providerValue(payload, "voice_speed"); ok {
		body["voice_speed"] = value
	} else if value, ok := providerValue(payload, "speed"); ok {
		body["voice_speed"] = value
	}
	return jsonProviderRequest(http.MethodPost, "/kling/v1/audio/tts", body)
}

func buildKlingCustomVoicesRequest(payload map[string]any) (providerProtocolRequest, error) {
	voiceID := firstProviderString(payload, "voice_id", "id", "task_id")
	if voiceID != "" {
		return providerProtocolRequest{
			Method:   http.MethodGet,
			Endpoint: "/kling/v1/general/custom-voices/" + url.PathEscape(voiceID),
		}, nil
	}
	voiceName := firstProviderString(payload, "voice_name", "name")
	voiceURL := firstProviderString(payload, "voice_url", "audio_url", "url")
	if voiceName == "" {
		return providerProtocolRequest{}, fmt.Errorf("voice_name is required")
	}
	if voiceURL == "" {
		return providerProtocolRequest{}, fmt.Errorf("voice_url is required")
	}
	return jsonProviderRequest(http.MethodPost, "/kling/v1/general/custom-voices", map[string]any{
		"voice_name": voiceName,
		"voice_url":  voiceURL,
	})
}

func buildKlingPresetVoicesRequest(payload map[string]any) (providerProtocolRequest, error) {
	pageNum := providerPositiveIntDefault(payload, 1, "pageNum", "page_num", "page")
	pageSize := providerPositiveIntDefault(payload, 5, "pageSize", "page_size", "limit")
	return providerProtocolRequest{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/kling/v1/general/presets-voices?pageNum=%d&pageSize=%d", pageNum, pageSize),
	}, nil
}

func jsonProviderRequest(method string, endpoint string, payload map[string]any) (providerProtocolRequest, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return providerProtocolRequest{}, err
	}
	return providerProtocolRequest{
		Method:      method,
		Endpoint:    endpoint,
		Body:        body,
		ContentType: "application/json",
	}, nil
}

func parseProviderProtocolPayload(body []byte, contentType string) (map[string]any, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, fmt.Errorf("multipart boundary is required")
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		form, err := reader.ReadForm(16 << 20)
		if err != nil {
			return nil, fmt.Errorf("read multipart body: %w", err)
		}
		defer func() { _ = form.RemoveAll() }()

		payload := make(map[string]any)
		for key, values := range form.Value {
			if len(values) == 1 {
				payload[key] = strings.TrimSpace(values[0])
				continue
			}
			items := make([]string, 0, len(values))
			for _, value := range values {
				items = append(items, strings.TrimSpace(value))
			}
			payload[key] = items
		}
		return payload, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil && err != io.EOF {
		return nil, fmt.Errorf("parse request body: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func firstProviderString(payload map[string]any, paths ...string) string {
	for _, path := range paths {
		if value := providerString(payload, path); value != "" {
			return value
		}
	}
	return ""
}

func providerStringDefault(payload map[string]any, fallback string, paths ...string) string {
	if value := firstProviderString(payload, paths...); value != "" {
		return value
	}
	return fallback
}

func providerString(payload map[string]any, path string) string {
	value, ok := providerValue(payload, path)
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func providerStringSlice(payload map[string]any, paths ...string) []string {
	for _, path := range paths {
		value, ok := providerValue(payload, path)
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if value := strings.TrimSpace(fmt.Sprint(item)); value != "" {
					out = append(out, value)
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if value := strings.TrimSpace(item); value != "" {
					out = append(out, value)
				}
			}
			return out
		case string:
			if value := strings.TrimSpace(typed); value != "" {
				return []string{value}
			}
		}
	}
	return nil
}

func providerPositiveIntDefault(payload map[string]any, fallback int, paths ...string) int {
	for _, path := range paths {
		value, ok := providerValue(payload, path)
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case json.Number:
			if parsed, err := strconv.Atoi(typed.String()); err == nil && parsed > 0 {
				return parsed
			}
		case float64:
			if typed > 0 {
				return int(typed)
			}
		case int:
			if typed > 0 {
				return typed
			}
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return fallback
}

func providerValue(payload map[string]any, path string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current any = payload
	for _, part := range parts {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
