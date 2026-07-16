//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleGrokMediaErrorResponseVideoStatus502DoesNotUnscheduleAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/request-123", nil)
	account := &Account{
		ID:          7101,
		Name:        "grok-video-status",
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	repo := &grokQuotaAccountRepo{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"temporary video status failure"}}`)),
	}

	result, err := svc.handleGrokMediaErrorResponse(
		context.Background(),
		resp,
		c,
		account,
		GrokMediaEndpointVideoStatus,
		"request-id",
		"",
	)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Zero(t, repo.tempUnschedCalls)
	require.Zero(t, repo.rateLimitedCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestHandleGrokMediaAccountUpstreamErrorPreservesNonStatus5xxBehavior(t *testing.T) {
	tests := []struct {
		name              string
		endpoint          GrokMediaEndpoint
		statusCode        int
		headers           http.Header
		wantTempUnsched   int
		wantRateLimited   int
		wantReason        string
		wantMinBlockAfter time.Duration
	}{
		{
			name:              "video generation 502 keeps upstream cooldown",
			endpoint:          GrokMediaEndpointVideosGenerations,
			statusCode:        http.StatusBadGateway,
			wantTempUnsched:   1,
			wantReason:        "grok upstream temporary error",
			wantMinBlockAfter: 2*time.Minute - time.Second,
		},
		{
			name:              "video status 401 keeps credential cooldown",
			endpoint:          GrokMediaEndpointVideoStatus,
			statusCode:        http.StatusUnauthorized,
			wantTempUnsched:   1,
			wantReason:        "grok credentials unauthorized",
			wantMinBlockAfter: 10*time.Minute - time.Second,
		},
		{
			name:              "video status 403 keeps entitlement cooldown",
			endpoint:          GrokMediaEndpointVideoStatus,
			statusCode:        http.StatusForbidden,
			wantTempUnsched:   1,
			wantReason:        "grok access or entitlement denied",
			wantMinBlockAfter: 30*time.Minute - time.Second,
		},
		{
			name:            "video status 429 keeps rate limit",
			endpoint:        GrokMediaEndpointVideoStatus,
			statusCode:      http.StatusTooManyRequests,
			headers:         http.Header{"Retry-After": []string{"45"}},
			wantRateLimited: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{ID: 7102, Platform: PlatformGrok, Type: AccountTypeOAuth}
			repo := &grokQuotaAccountRepo{}
			svc := &OpenAIGatewayService{accountRepo: repo}
			before := time.Now()

			svc.handleGrokMediaAccountUpstreamError(
				context.Background(),
				account,
				tt.endpoint,
				tt.statusCode,
				tt.headers,
				nil,
			)

			require.Equal(t, tt.wantTempUnsched, repo.tempUnschedCalls)
			require.Equal(t, tt.wantRateLimited, repo.rateLimitedCalls)
			if tt.wantTempUnsched > 0 {
				require.Equal(t, tt.wantReason, repo.lastTempUnschedReason)
				require.True(t, repo.lastTempUnschedUntil.After(before.Add(tt.wantMinBlockAfter)))
				require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
			}
			if tt.wantRateLimited > 0 {
				require.WithinDuration(t, before.Add(45*time.Second), repo.lastRateLimitResetAt, time.Second)
				require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
			}
		})
	}
}
