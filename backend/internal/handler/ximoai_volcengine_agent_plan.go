package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *GatewayHandler) VolcengineAgentPlanImages(c *gin.Context) {
	h.handleVolcengineAgentPlan(c, service.VolcengineAgentPlanImagesGenerations)
}

func (h *GatewayHandler) VolcengineAgentPlanTTSUnidirectional(c *gin.Context) {
	h.handleVolcengineAgentPlan(c, service.VolcengineAgentPlanTTSUnidirectional)
}

func (h *GatewayHandler) VolcengineAgentPlanTTSUnidirectionalStream(c *gin.Context) {
	h.handleVolcengineAgentPlan(c, service.VolcengineAgentPlanTTSUnidirectionalStream)
}

func (h *GatewayHandler) VolcengineAgentPlanTTSBidirection(c *gin.Context) {
	h.handleVolcengineAgentPlan(c, service.VolcengineAgentPlanTTSBidirection)
}

func (h *GatewayHandler) VolcengineAgentPlanASRBigmodelAsync(c *gin.Context) {
	h.handleVolcengineAgentPlan(c, service.VolcengineAgentPlanASRBigmodelAsync)
}

func (h *GatewayHandler) VolcengineAgentPlanASRBigmodelNostream(c *gin.Context) {
	h.handleVolcengineAgentPlan(c, service.VolcengineAgentPlanASRBigmodelNostream)
}

func (h *GatewayHandler) handleVolcengineAgentPlan(c *gin.Context, endpoint service.VolcengineAgentPlanEndpoint) {
	apiKey, ok := servermiddleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	if apiKey.Group == nil || h.XimoAIPlatformKind(c.Request.Context(), apiKey.Group.Platform) != service.PlatformKindVolcengineAgentPlan {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Volcengine Agent Plan API is not supported for this platform")
		return
	}

	body, requestedModel, ok := h.parseVolcengineAgentPlanRequest(c, endpoint)
	if !ok {
		return
	}
	if endpoint == service.VolcengineAgentPlanImagesGenerations && !service.GroupAllowsImageGeneration(apiKey.Group) {
		h.errorResponse(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
		return
	}
	if h.gatewayService == nil || h.billingCacheService == nil || h.concurrencyHelper == nil {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "Gateway dependencies are unavailable")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.gateway.volcengine_agent_plan",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.String("endpoint", string(endpoint)),
		zap.String("model", requestedModel),
	)
	setOpsRequestContext(c, requestedModel, endpoint.IsWebSocket())
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(endpoint.IsWebSocket(), false)))

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, requestedModel)
	subscription, _ := servermiddleware.GetSubscriptionFromContext(c)
	streamStarted := false
	userRelease, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		h.handleConcurrencyError(c, err, "user", false)
		return
	}
	userRelease = wrapReleaseOnDone(c.Request.Context(), userRelease)
	if userRelease != nil {
		defer userRelease()
	}

	if err := h.billingCacheService.CheckBillingEligibility(
		c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey),
	); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, false)
		return
	}

	failedAccountIDs := make(map[int64]struct{})
	maxSwitches := h.maxAccountSwitches
	if maxSwitches <= 0 {
		maxSwitches = 10
	}
	var lastErr error
	for switchCount := 0; switchCount <= maxSwitches; switchCount++ {
		selection, selectErr := h.gatewayService.SelectAccountWithLoadAwareness(
			c.Request.Context(), apiKey.GroupID, "", requestedModel, failedAccountIDs, "", subject.UserID,
		)
		if selectErr != nil {
			if lastErr != nil {
				h.writeVolcengineAgentPlanError(c, lastErr)
				return
			}
			markOpsRoutingCapacityLimitedIfNoAvailable(c, selectErr)
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Volcengine Agent Plan accounts")
			return
		}
		if selection == nil || selection.Account == nil {
			markOpsRoutingCapacityLimited(c)
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Volcengine Agent Plan accounts")
			return
		}

		account := selection.Account
		if account.Type != service.AccountTypeAPIKey || account.PlatformRuntimeKind() != service.PlatformKindVolcengineAgentPlan {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			failedAccountIDs[account.ID] = struct{}{}
			lastErr = errors.New("selected account is not a Volcengine Agent Plan API Key account")
			continue
		}
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountRelease, acquired := h.acquireVolcengineAgentPlanAccountSlot(c, selection, &streamStarted, reqLog)
		if !acquired {
			return
		}

		upstreamModel := requestedModel
		if endpoint == service.VolcengineAgentPlanImagesGenerations {
			upstreamModel = strings.TrimSpace(channelMapping.MappedModel)
			if upstreamModel == "" {
				upstreamModel = requestedModel
			}
			upstreamModel = account.GetMappedModel(upstreamModel)
		}

		var result *service.ForwardResult
		if endpoint.IsWebSocket() {
			result, err = h.gatewayService.ProxyVolcengineAgentPlanWebSocket(c.Request.Context(), c, account, endpoint)
		} else {
			result, err = h.gatewayService.ForwardVolcengineAgentPlanHTTP(c.Request.Context(), c, account, endpoint, body, upstreamModel)
		}
		if accountRelease != nil {
			accountRelease()
		}

		if err != nil {
			lastErr = err
			if c.Writer.Written() {
				reqLog.Warn("volcengine_agent_plan.forward_failed_after_response_started", zap.Int64("account_id", account.ID), zap.Error(err))
				return
			}
			if switchCount < maxSwitches && retryVolcengineAgentPlanWithAnotherAccount(err) {
				failedAccountIDs[account.ID] = struct{}{}
				continue
			}
			h.writeVolcengineAgentPlanError(c, err)
			return
		}

		result.Model = requestedModel
		if strings.TrimSpace(result.UpstreamModel) == "" {
			result.UpstreamModel = upstreamModel
		}
		h.recordVolcengineAgentPlanUsage(c, reqLog, apiKey, subject, subscription, account, result, body, endpoint, channelMapping)
		return
	}

	h.writeVolcengineAgentPlanError(c, lastErr)
}

func (h *GatewayHandler) parseVolcengineAgentPlanRequest(
	c *gin.Context,
	endpoint service.VolcengineAgentPlanEndpoint,
) ([]byte, string, bool) {
	if endpoint.IsWebSocket() {
		return nil, endpoint.BillingModel(), true
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
		} else {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		}
		return nil, "", false
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return nil, "", false
	}
	if endpoint != service.VolcengineAgentPlanImagesGenerations {
		return body, endpoint.BillingModel(), true
	}
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body must be valid JSON")
		return nil, "", false
	}
	modelValue := gjson.GetBytes(body, "model")
	model := strings.TrimSpace(modelValue.String())
	if !modelValue.Exists() || modelValue.Type != gjson.String || model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, "", false
	}
	return body, model, true
}

func (h *GatewayHandler) acquireVolcengineAgentPlanAccountSlot(
	c *gin.Context,
	selection *service.AccountSelectionResult,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	if selection.Acquired {
		return wrapReleaseOnDone(c.Request.Context(), selection.ReleaseFunc), true
	}
	if selection.WaitPlan == nil {
		markOpsRoutingCapacityLimited(c)
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Volcengine Agent Plan accounts")
		return nil, false
	}

	accountID := selection.Account.ID
	canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), accountID, selection.WaitPlan.MaxWaiting)
	if err != nil {
		reqLog.Warn("volcengine_agent_plan.account_wait_counter_failed", zap.Int64("account_id", accountID), zap.Error(err))
	} else if !canWait {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return nil, false
	}
	if err == nil && canWait {
		defer h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), accountID)
	}

	release, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
		c,
		accountID,
		selection.WaitPlan.MaxConcurrency,
		selection.WaitPlan.Timeout,
		false,
		streamStarted,
	)
	if err != nil {
		h.handleConcurrencyError(c, err, "account", false)
		return nil, false
	}
	return wrapReleaseOnDone(c.Request.Context(), release), true
}

func retryVolcengineAgentPlanWithAnotherAccount(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var upstreamErr *service.VolcengineAgentPlanUpstreamError
	if errors.As(err, &upstreamErr) {
		return upstreamErr.Retryable()
	}
	return true
}

func (h *GatewayHandler) writeVolcengineAgentPlanError(c *gin.Context, err error) {
	if c.Writer.Written() {
		return
	}
	var upstreamErr *service.VolcengineAgentPlanUpstreamError
	if errors.As(err, &upstreamErr) {
		for _, header := range []string{"X-Api-Status-Code", "X-Api-Message", "X-Tt-Logid", "X-Request-Id"} {
			if value := strings.TrimSpace(upstreamErr.Headers.Get(header)); value != "" {
				c.Header(header, value)
			}
		}
		contentType := strings.TrimSpace(upstreamErr.Headers.Get("Content-Type"))
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(upstreamErr.StatusCode, contentType, upstreamErr.Body)
		return
	}
	h.errorResponse(c, http.StatusBadGateway, "api_error", "Volcengine Agent Plan upstream request failed")
}

func (h *GatewayHandler) recordVolcengineAgentPlanUsage(
	c *gin.Context,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subject servermiddleware.AuthSubject,
	subscription *service.UserSubscription,
	account *service.Account,
	result *service.ForwardResult,
	body []byte,
	endpoint service.VolcengineAgentPlanEndpoint,
	channelMapping service.ChannelMappingResult,
) {
	if result == nil {
		return
	}
	upstreamEndpoint := ""
	if rawURL, err := service.VolcengineAgentPlanUpstreamURL(endpoint); err == nil {
		if parsed, parseErr := url.Parse(rawURL); parseErr == nil {
			upstreamEndpoint = parsed.Path
		}
	}
	requestPayloadHash := service.HashUsageRequestPayload(body)
	if len(body) == 0 {
		requestPayloadHash = service.HashUsageRequestPayload([]byte(endpoint))
	}
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	usageFields := channelMapping.ToUsageFields(result.Model, result.UpstreamModel)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)

	h.submitUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
			Result:             result,
			QuotaPlatform:      quotaPlatform,
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    inboundEndpoint,
			UpstreamEndpoint:   upstreamEndpoint,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
			ChannelUsageFields: usageFields,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.gateway.volcengine_agent_plan"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Int64("account_id", account.ID),
				zap.String("model", result.Model),
			).Error("volcengine_agent_plan.record_usage_failed", zap.Error(err))
		}
	})
}
