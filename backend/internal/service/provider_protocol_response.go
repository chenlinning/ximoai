package service

import (
	"encoding/json"
	"strings"
)

func adaptOpenAIVideoProviderResponse(account *Account, endpoint string, body []byte) ([]byte, error) {
	if account == nil || !account.IsGrokVideo() || len(body) == 0 {
		return body, nil
	}

	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	endpoint = normalizeOpenAIVideoEndpointPath(endpoint)
	if !isGrokVideoMutationEndpoint(endpoint) {
		return body, nil
	}
	if providerString(payload, "request_id") == "" {
		if requestID := firstProviderString(payload, "id", "task_id", "data.id", "data.task_id"); requestID != "" {
			payload["request_id"] = requestID
		}
	}
	return json.Marshal(payload)
}

func isGrokVideoMutationEndpoint(endpoint string) bool {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	return strings.HasSuffix(endpoint, "/generations") ||
		strings.HasSuffix(endpoint, "/edits") ||
		strings.HasSuffix(endpoint, "/extensions")
}
