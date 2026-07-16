package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) VideosCreate(c *gin.Context) {
	h.handleVideoMutation(c, "", false)
}

func (h *OpenAIGatewayHandler) VideosSubpath(c *gin.Context) {
	subpath := strings.Trim(strings.TrimSpace(c.Param("subpath")), "/")
	parts := splitOpenAIVideoSubpath(subpath)
	if len(parts) == 0 {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video endpoint not found")
		return
	}

	switch c.Request.Method {
	case http.MethodPost:
		switch {
		case len(parts) == 1 && (parts[0] == "edits" || parts[0] == "extensions"):
			h.handleVideoMutation(c, "", false)
			return
		case len(parts) == 1 && parts[0] == "characters":
			h.handleVideoCharacterCreate(c)
			return
		case len(parts) == 2 && parts[1] == "remix":
			h.handleVideoMutation(c, parts[0], false)
			return
		}
	case http.MethodGet:
		switch {
		case len(parts) == 1:
			h.handleVideoRead(c, http.MethodGet, false, parts[0])
			return
		case len(parts) == 2 && parts[1] == "content":
			h.handleVideoRead(c, http.MethodGet, true, parts[0])
			return
		case len(parts) == 2 && parts[0] == "characters":
			h.handleVideoCharacterRead(c, parts[1])
			return
		}
	case http.MethodDelete:
		if len(parts) == 1 {
			h.handleVideoRead(c, http.MethodDelete, false, parts[0])
			return
		}
	}

	h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video endpoint not found")
}

func (h *OpenAIGatewayHandler) VideosRemix(c *gin.Context) {
	h.handleVideoMutation(c, strings.TrimSpace(c.Param("id")), false)
}

func (h *OpenAIGatewayHandler) VideosRetrieve(c *gin.Context) {
	h.handleVideoRead(c, http.MethodGet, false)
}

func (h *OpenAIGatewayHandler) VideosDelete(c *gin.Context) {
	h.handleVideoRead(c, http.MethodDelete, false)
}

func (h *OpenAIGatewayHandler) VideosContent(c *gin.Context) {
	h.handleVideoRead(c, http.MethodGet, true)
}

func (h *OpenAIGatewayHandler) handleVideoCharacterCreate(c *gin.Context) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if isMultipartVideoContentType(c.GetHeader("Content-Type")) {
		setOpsRequestContext(c, "", false)
	} else {
		setOpsRequestContext(c, "", false)
	}
	parsed, err := h.gatewayService.ParseOpenAIVideoRequest(c, body, false)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	var fixedAccount *service.Account
	for _, id := range uniqueOpenAIVideoIDs(parsed.SourceVideoIDs) {
		job, account, resolveErr := h.gatewayService.ResolveOpenAIVideoJobAccount(c.Request.Context(), id)
		if resolveErr != nil || !canAccessOpenAIVideoJob(job, apiKey, subject, openAIPlatformForAPIKey(apiKey)) {
			h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
			return
		}
		if fixedAccount != nil && account != nil && fixedAccount.ID != account.ID {
			h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
			return
		}
		fixedAccount = account
	}

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai.videos.character_billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	meta := service.OpenAIVideoJobMeta{
		Platform: openAIPlatformForAPIKey(apiKey),
		GroupID:  apiKey.GroupID,
		APIKeyID: int64Ptr(apiKey.ID),
		UserID:   int64Ptr(subject.UserID),
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	if fixedAccount != nil {
		setOpsSelectedAccount(c, fixedAccount.ID, fixedAccount.Platform)
		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		_, err := h.gatewayService.ForwardOpenAIVideoCharacterCreate(c.Request.Context(), c, fixedAccount, body, c.GetHeader("Content-Type"), meta)
		h.setOpenAIResponseLatency(c, time.Since(forwardStart).Milliseconds())
		if err != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(fixedAccount.ID, "", false, nil)
			wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
			reqLog.Warn("openai.videos.character_fixed_account_forward_failed", zap.Int64("account_id", fixedAccount.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
			return
		}
		h.gatewayService.ReportOpenAIAccountScheduleResult(fixedAccount.ID, "", true, nil)
		return
	}

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		selection, _, err := h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(),
			openAIPlatformForAPIKey(apiKey),
			apiKey.GroupID,
			"",
			sessionHash,
			"",
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			false,
		)
		if err != nil {
			reqLog.Warn("openai.videos.character_account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
			if len(failedAccountIDs) == 0 {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible API Key accounts", streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
			} else {
				h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible API Key accounts", streamStarted)
			return
		}
		account := selection.Account
		if account.Type != service.AccountTypeAPIKey {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			failedAccountIDs[account.ID] = struct{}{}
			continue
		}
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		setOpsSelectedAccount(c, account.ID, account.Platform)
		if account.Type != service.AccountTypeAPIKey {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, "", false, nil)
			failedAccountIDs[account.ID] = struct{}{}
			if switchCount >= maxAccountSwitches {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible API Key accounts", streamStarted)
				return
			}
			switchCount++
			continue
		}

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		_, err = h.gatewayService.ForwardOpenAIVideoCharacterCreate(c.Request.Context(), c, account, body, c.GetHeader("Content-Type"), meta)
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		h.setOpenAIResponseLatency(c, forwardDurationMs)

		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, "", false, nil)
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						select {
						case <-c.Request.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, "", false, nil)
			wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
			reqLog.Warn("openai.videos.character_forward_failed", zap.Int64("account_id", account.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, "", true, nil)
		reqLog.Debug("openai.videos.character_request_completed", zap.Int64("account_id", account.ID), zap.Int("switch_count", switchCount))
		return
	}
}

func (h *OpenAIGatewayHandler) handleVideoMutation(c *gin.Context, sourceVideoID string, requireModel bool) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	parsed, err := h.gatewayService.ParseOpenAIVideoRequest(c, body, requireModel)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	var fixedAccount *service.Account
	sourceVideoIDs := uniqueOpenAIVideoIDs(append([]string{sourceVideoID}, parsed.SourceVideoIDs...))
	for _, id := range sourceVideoIDs {
		job, account, resolveErr := h.gatewayService.ResolveOpenAIVideoJobAccount(c.Request.Context(), id)
		if resolveErr != nil || !canAccessOpenAIVideoJob(job, apiKey, subject, openAIPlatformForAPIKey(apiKey)) {
			h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
			return
		}
		if fixedAccount != nil && account != nil && fixedAccount.ID != account.ID {
			h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
			return
		}
		fixedAccount = account
		if parsed.Model == "" {
			parsed.Model = strings.TrimSpace(job.RequestModel)
		}
	}

	reqLog = reqLog.With(zap.String("model", parsed.Model), zap.String("source_video_id", sourceVideoID))
	setOpsRequestContext(c, parsed.Model, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, parsed.Model, body); decision != nil && decision.Blocked {
		h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
		return
	}

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, parsed.Model)
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai.videos.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	meta := service.OpenAIVideoJobMeta{
		Platform:  openAIPlatformForAPIKey(apiKey),
		GroupID:   apiKey.GroupID,
		APIKeyID:  int64Ptr(apiKey.ID),
		UserID:    int64Ptr(subject.UserID),
		ChannelID: positiveInt64Ptr(channelMapping.ChannelID),
	}

	if fixedAccount != nil {
		setOpsSelectedAccount(c, fixedAccount.ID, fixedAccount.Platform)
		h.forwardVideoMutationWithAccount(c, reqLog, apiKey, subject, subscription, fixedAccount, body, parsed, channelMapping, meta, routingStart, &streamStarted, 0, nil)
		return
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(),
			openAIPlatformForAPIKey(apiKey),
			apiKey.GroupID,
			"",
			sessionHash,
			parsed.Model,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			false,
		)
		if err != nil {
			reqLog.Warn("openai.videos.account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
			if len(failedAccountIDs) == 0 {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible API Key accounts", streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
			} else {
				h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible API Key accounts", streamStarted)
			return
		}
		_ = scheduleDecision
		account := selection.Account
		if account.Type != service.AccountTypeAPIKey {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			failedAccountIDs[account.ID] = struct{}{}
			continue
		}
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		setOpsSelectedAccount(c, account.ID, account.Platform)
		scheduledModel := account.GetMappedModel(openAIChannelRoutingModel(parsed.Model, channelMapping))

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		result, err := h.gatewayService.ForwardOpenAIVideoMutation(c.Request.Context(), c, account, body, parsed, channelMapping.MappedModel, meta)
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		h.setOpenAIResponseLatency(c, forwardDurationMs)

		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, scheduledModel, false, nil)
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						select {
						case <-c.Request.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, scheduledModel, false, nil)
			wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
			reqLog.Warn("openai.videos.forward_failed", zap.Int64("account_id", account.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, scheduledModel, true, nil)
		h.recordVideoUsage(c, reqLog, apiKey, subject, subscription, account, result, body, parsed.Model, channelMapping)
		reqLog.Debug("openai.videos.request_completed", zap.Int64("account_id", account.ID), zap.Int("switch_count", switchCount))
		return
	}
}

func (h *OpenAIGatewayHandler) forwardVideoMutationWithAccount(
	c *gin.Context,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subject middleware2.AuthSubject,
	subscription *service.UserSubscription,
	account *service.Account,
	body []byte,
	parsed *service.OpenAIVideoRequest,
	channelMapping service.ChannelMappingResult,
	meta service.OpenAIVideoJobMeta,
	routingStart time.Time,
	streamStarted *bool,
	switchCount int,
	_ *service.OpenAIVideoJob,
) {
	scheduledModel := account.GetMappedModel(openAIChannelRoutingModel(parsed.Model, channelMapping))
	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
	forwardStart := time.Now()
	result, err := h.gatewayService.ForwardOpenAIVideoMutation(c.Request.Context(), c, account, body, parsed, channelMapping.MappedModel, meta)
	forwardDurationMs := time.Since(forwardStart).Milliseconds()
	h.setOpenAIResponseLatency(c, forwardDurationMs)
	if err != nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, scheduledModel, false, nil)
		wroteFallback := h.ensureForwardErrorResponse(c, *streamStarted)
		reqLog.Warn("openai.videos.forward_fixed_account_failed",
			zap.Int64("account_id", account.ID),
			zap.Bool("fallback_error_response_written", wroteFallback),
			zap.Error(err),
		)
		return
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, scheduledModel, true, nil)
	h.recordVideoUsage(c, reqLog, apiKey, subject, subscription, account, result, body, parsed.Model, channelMapping)
	reqLog.Debug("openai.videos.request_completed", zap.Int64("account_id", account.ID), zap.Int("switch_count", switchCount))
}

func (h *OpenAIGatewayHandler) handleVideoRead(c *gin.Context, method string, content bool, explicitVideoID ...string) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	videoID := strings.TrimSpace(c.Param("id"))
	if len(explicitVideoID) > 0 {
		videoID = strings.TrimSpace(explicitVideoID[0])
	}
	if videoID == "" && strings.EqualFold(method, http.MethodGet) && !content {
		limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
		jobs, err := h.gatewayService.ListOpenAIVideoJobs(c.Request.Context(), service.OpenAIVideoJobListFilter{
			Platform: openAIPlatformForAPIKey(apiKey),
			GroupID:  apiKey.GroupID,
			UserID:   &subject.UserID,
			After:    c.Query("after"),
			Limit:    limit,
			Order:    c.Query("order"),
		})
		if err != nil {
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to list videos")
			return
		}
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": jobs})
		return
	}

	var account *service.Account
	if videoID != "" {
		job, jobAccount, err := h.gatewayService.ResolveOpenAIVideoJobAccount(c.Request.Context(), videoID)
		if err != nil || !canAccessOpenAIVideoJob(job, apiKey, subject, openAIPlatformForAPIKey(apiKey)) {
			h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
			return
		}
		account = jobAccount
	}
	if account == nil {
		selection, _, err := h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(),
			openAIPlatformForAPIKey(apiKey),
			apiKey.GroupID,
			"",
			"",
			"",
			nil,
			service.OpenAIUpstreamTransportHTTPSSE,
			false,
		)
		if err != nil || selection == nil || selection.Account == nil || selection.Account.Type != service.AccountTypeAPIKey {
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible API Key accounts", streamStarted)
			return
		}
		account = selection.Account
		if selection.ReleaseFunc != nil {
			defer selection.ReleaseFunc()
		}
	}
	setOpsSelectedAccount(c, account.ID, account.Platform)

	endpoint := c.Request.URL.Path
	if content {
		if err := h.gatewayService.ForwardOpenAIVideoContent(c.Request.Context(), c, account, endpoint); err != nil {
			wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
			reqLog.Warn("openai.videos.content_forward_failed", zap.Int64("account_id", account.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
		}
		return
	}
	if _, err := h.gatewayService.ForwardOpenAIVideoJSON(c.Request.Context(), c, account, method, endpoint); err != nil {
		wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
		reqLog.Warn("openai.videos.json_forward_failed", zap.Int64("account_id", account.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
	}
}

func (h *OpenAIGatewayHandler) handleVideoCharacterRead(c *gin.Context, characterID string) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video character not found")
		return
	}
	character, account, err := h.gatewayService.ResolveOpenAIVideoCharacterAccount(c.Request.Context(), characterID)
	if err != nil || !canAccessOpenAIVideoCharacter(character, apiKey, subject, openAIPlatformForAPIKey(apiKey)) {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video character not found")
		return
	}
	setOpsSelectedAccount(c, account.ID, account.Platform)

	if _, err := h.gatewayService.ForwardOpenAIVideoCharacterJSON(c.Request.Context(), c, account, http.MethodGet, c.Request.URL.Path); err != nil {
		wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
		reqLog.Warn("openai.videos.character_json_forward_failed", zap.Int64("account_id", account.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
	}
}

func (h *OpenAIGatewayHandler) recordVideoUsage(
	c *gin.Context,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subject middleware2.AuthSubject,
	subscription *service.UserSubscription,
	account *service.Account,
	result *service.OpenAIForwardResult,
	body []byte,
	reqModel string,
	channelMapping service.ChannelMappingResult,
) {
	if result == nil {
		return
	}
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	requestPayloadHash := service.HashUsageRequestPayload(body)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             result,
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
			ChannelUsageFields: channelMapping.ToUsageFields(reqModel, result.UpstreamModel),
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.videos"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", reqModel),
				zap.Int64("account_id", account.ID),
			).Error("openai.videos.record_usage_failed", zap.Error(err))
		}
	})
}

func (h *OpenAIGatewayHandler) setOpenAIResponseLatency(c *gin.Context, forwardDurationMs int64) {
	upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
	responseLatencyMs := forwardDurationMs
	if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
		responseLatencyMs = forwardDurationMs - upstreamLatencyMs
	}
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
}

func int64Ptr(v int64) *int64 {
	return &v
}

func positiveInt64Ptr(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func splitOpenAIVideoSubpath(subpath string) []string {
	subpath = strings.Trim(strings.TrimSpace(subpath), "/")
	if subpath == "" {
		return nil
	}
	rawParts := strings.Split(subpath, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func uniqueOpenAIVideoIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func isMultipartVideoContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data")
}

func canAccessOpenAIVideoJob(job *service.OpenAIVideoJob, apiKey *service.APIKey, subject middleware2.AuthSubject, platform string) bool {
	if job == nil || apiKey == nil {
		return false
	}
	platform = service.NormalizePlatformSlug(platform)
	if job.Platform != "" && platform != "" && service.NormalizePlatformSlug(job.Platform) != platform {
		return false
	}
	if job.GroupID != nil {
		if apiKey.GroupID == nil || *job.GroupID != *apiKey.GroupID {
			return false
		}
	}
	if job.UserID != nil {
		return *job.UserID == subject.UserID
	}
	if job.APIKeyID != nil {
		return *job.APIKeyID == apiKey.ID
	}
	return false
}

func canAccessOpenAIVideoCharacter(character *service.OpenAIVideoCharacter, apiKey *service.APIKey, subject middleware2.AuthSubject, platform string) bool {
	if character == nil || apiKey == nil {
		return false
	}
	platform = service.NormalizePlatformSlug(platform)
	if character.Platform != "" && platform != "" && service.NormalizePlatformSlug(character.Platform) != platform {
		return false
	}
	if character.GroupID != nil {
		if apiKey.GroupID == nil || *character.GroupID != *apiKey.GroupID {
			return false
		}
	}
	if character.UserID != nil {
		return *character.UserID == subject.UserID
	}
	if character.APIKeyID != nil {
		return *character.APIKeyID == apiKey.ID
	}
	return false
}
