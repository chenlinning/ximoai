package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *AccountTestService) testVolcengineAgentPlanConnection(
	c *gin.Context,
	account *Account,
	modelID string,
	prompt string,
) error {
	model := strings.TrimSpace(modelID)
	if model == "" {
		model = VolcengineAgentPlanSeedreamModel
	}
	if model != VolcengineAgentPlanSeedreamModel {
		return s.sendErrorAndEnd(c, "Account connection test supports Seedream only; use the native TTS or ASR endpoint for audio verification")
	}
	if account == nil || account.Type != AccountTypeAPIKey {
		return s.sendErrorAndEnd(c, "Volcengine Agent Plan requires an API Key account")
	}
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = defaultOpenAIImageTestPrompt
	}

	upstreamModel := account.GetMappedModel(model)
	payload, err := json.Marshal(map[string]any{
		"model":  upstreamModel,
		"prompt": prompt,
		"size":   "1024x1024",
	})
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create test payload")
	}
	targetURL, _ := VolcengineAgentPlanUpstreamURL(VolcengineAgentPlanImagesGenerations)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()
	s.sendEvent(c, TestEvent{Type: "test_start", Model: model})

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var resp *http.Response
	if s.tlsFPProfileService != nil {
		profile := s.tlsFPProfileService.ResolveTLSProfile(account)
		if profile != nil {
			resp, err = s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, profile)
		}
	}
	if resp == nil && err == nil {
		resp, err = s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	}
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	if resp == nil {
		return s.sendErrorAndEnd(c, "Upstream returned no response")
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readVolcengineAgentPlanResponseBody(resp.Body)
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}
	providerCode := strings.TrimSpace(resp.Header.Get("X-Api-Status-Code"))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices ||
		(providerCode != "" && !volcengineAgentPlanSuccessCode(providerCode)) {
		return s.sendErrorAndEnd(c, fmt.Sprintf("API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	imageURL := strings.TrimSpace(gjson.GetBytes(body, "data.0.url").String())
	if imageURL == "" {
		if encoded := strings.TrimSpace(gjson.GetBytes(body, "data.0.b64_json").String()); encoded != "" {
			imageURL = "data:image/png;base64," + encoded
		}
	}
	if imageURL != "" {
		s.sendEvent(c, TestEvent{Type: "image", ImageURL: imageURL, MimeType: "image/png"})
	} else {
		s.sendEvent(c, TestEvent{Type: "content", Text: "Seedream request completed successfully"})
	}
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
