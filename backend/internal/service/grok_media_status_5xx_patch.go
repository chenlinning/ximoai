package service

import (
	"context"
	"net/http"
	"time"
)

func (s *OpenAIGatewayService) handleGrokMediaAccountUpstreamError(
	ctx context.Context,
	account *Account,
	endpoint GrokMediaEndpoint,
	statusCode int,
	headers http.Header,
	responseBody []byte,
) {
	// Temporary upstream bug fix: video status polling can intermittently return
	// 5xx and must not evict the whole Grok account. Remove this endpoint-specific
	// branch when upstream provides equivalent scheduling behavior.
	if endpoint == GrokMediaEndpointVideoStatus && statusCode >= 500 {
		if s == nil || account == nil {
			return
		}
		s.updateGrokUsageSnapshot(ctx, account, parseGrokQuotaSnapshot(headers, statusCode, time.Now()))
		return
	}

	s.handleGrokAccountUpstreamError(ctx, account, statusCode, headers, responseBody)
}
