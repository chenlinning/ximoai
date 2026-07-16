package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkgerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const gatewayCompatibilityMetricsLogInterval = 1024

var gatewayCompatibilityMetricsLogCounter atomic.Uint64

// GatewayHandler handles API gateway requests
type GatewayHandler struct {
	gatewayService            *service.GatewayService
	openAIGatewayService      *service.OpenAIGatewayService
	geminiCompatService       *service.GeminiMessagesCompatService
	antigravityGatewayService *service.AntigravityGatewayService
	userService               *service.UserService
	billingCacheService       *service.BillingCacheService
	usageService              *service.UsageService
	apiKeyService             *service.APIKeyService
	usageRecordWorkerPool     *service.UsageRecordWorkerPool
	errorPassthroughService   *service.ErrorPassthroughService
	contentModerationService  *service.ContentModerationService
	platformService           *service.PlatformService
	concurrencyHelper         *ConcurrencyHelper
	userMsgQueueHelper        *UserMsgQueueHelper
	maxAccountSwitches        int
	maxAccountSwitchesGemini  int
	cfg                       *config.Config
	settingService            *service.SettingService
}

// NewGatewayHandler creates a new GatewayHandler
func NewGatewayHandler(
	gatewayService *service.GatewayService,
	openAIGatewayService *service.OpenAIGatewayService,
	geminiCompatService *service.GeminiMessagesCompatService,
	antigravityGatewayService *service.AntigravityGatewayService,
	userService *service.UserService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	platformService *service.PlatformService,
	userMsgQueueService *service.UserMessageQueueService,
	cfg *config.Config,
	settingService *service.SettingService,
) *GatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 10
	maxAccountSwitchesGemini := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
		if cfg.Gateway.MaxAccountSwitchesGemini > 0 {
			maxAccountSwitchesGemini = cfg.Gateway.MaxAccountSwitchesGemini
		}
	}

	// 初始化用户消息串行队列 helper
	var umqHelper *UserMsgQueueHelper
	if userMsgQueueService != nil && cfg != nil {
		umqHelper = NewUserMsgQueueHelper(userMsgQueueService, SSEPingFormatClaude, pingInterval)
	}

	return &GatewayHandler{
		gatewayService:            gatewayService,
		openAIGatewayService:      openAIGatewayService,
		geminiCompatService:       geminiCompatService,
		antigravityGatewayService: antigravityGatewayService,
		userService:               userService,
		billingCacheService:       billingCacheService,
		usageService:              usageService,
		apiKeyService:             apiKeyService,
		usageRecordWorkerPool:     usageRecordWorkerPool,
		errorPassthroughService:   errorPassthroughService,
		contentModerationService:  contentModerationService,
		platformService:           platformService,
		concurrencyHelper:         NewConcurrencyHelper(concurrencyService, SSEPingFormatClaude, pingInterval),
		userMsgQueueHelper:        umqHelper,
		maxAccountSwitches:        maxAccountSwitches,
		maxAccountSwitchesGemini:  maxAccountSwitchesGemini,
		cfg:                       cfg,
		settingService:            settingService,
	}
}

// Messages handles Claude API compatible messages endpoint
// POST /v1/messages
func (h *GatewayHandler) Messages(c *gin.Context) {
	// 从context获取apiKey和user（ApiKeyAuth中间件已设置）
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
		"handler.gateway.messages",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	defer h.maybeLogCompatibilityFallbackMetrics(reqLog)

	// 读取请求体
	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
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

	setOpsRequestContext(c, "", false)

	bodyRef := service.NewRequestBodyRef(body)
	parsedReq, err := service.ParseGatewayRequest(bodyRef, domain.PlatformAnthropic)
	if err != nil {
		logRequestBodyParseFailure(reqLog, body, err)
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	reqModel := parsedReq.Model
	reqStream := parsedReq.Stream
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	// 解析渠道级模型映射
	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, reqModel)

	// 设置 max_tokens=1 + haiku 探测请求标识到 context 中
	// 必须在 SetClaudeCodeClientContext 之前设置，因为 ClaudeCodeValidator 需要读取此标识进行绕过判断
	if isMaxTokensOneHaikuRequest(reqModel, parsedReq.MaxTokens) {
		ctx := service.WithIsMaxTokensOneHaikuRequest(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}

	// 检查是否为 Claude Code 客户端，设置到 context 中（复用已解析请求，避免二次反序列化）。
	SetClaudeCodeClientContext(c, body, parsedReq)
	isClaudeCodeClient := service.IsClaudeCodeClient(c.Request.Context())

	// 版本检查：仅对 Claude Code 客户端，拒绝低于最低版本的请求
	if !h.checkClaudeCodeVersion(c) {
		return
	}

	// 在请求上下文中记录 thinking 状态，供 Antigravity 最终模型 key 推导/模型维度限流使用
	c.Request = c.Request.WithContext(service.WithThinkingEnabled(c.Request.Context(), parsedReq.ThinkingEnabled, h.metadataBridgeEnabled()))

	setOpsRequestContext(c, reqModel, reqStream)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(reqStream, false)))

	// 验证 model 必填
	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolAnthropicMessages, reqModel, body); decision != nil && decision.Blocked {
		h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
		return
	}

	// Track if we've started streaming (for error handling)
	streamStarted := false

	// 绑定错误透传服务，允许 service 层在非 failover 错误场景复用规则。
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	// 获取订阅信息（可能为nil）- 提前获取用于后续检查
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	// 1. 首先获取用户并发槽位
	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted)
	if err != nil {
		reqLog.Warn("gateway.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", streamStarted)
		return
	}
	// 在请求结束或 Context 取消时确保释放槽位，避免客户端断开造成泄漏
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// 2. 【新增】Wait后二次检查余额/订阅
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("gateway.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	// 设置请求所属分组 ID（用于渠道级功能判断，如 WebSearch 模拟）
	parsedReq.GroupID = apiKey.GroupID

	// 计算粘性会话hash
	parsedReq.SessionContext = &service.SessionContext{
		ClientIP:  ip.GetClientIP(c),
		UserAgent: c.GetHeader("User-Agent"),
		APIKeyID:  apiKey.ID,
	}
	sessionHash := h.gatewayService.GenerateSessionHash(parsedReq)

	// [DEBUG-STICKY] 打印会话 hash 生成结果
	reqLog.Info("sticky.session_hash_generated",
		zap.String("session_hash", sessionHash),
		zap.String("metadata_user_id_raw", parsedReq.MetadataUserID),
	)

	// 获取平台：优先使用强制平台（/antigravity 路由，中间件已设置 request.Context），否则使用分组平台
	platform := ""
	if forcePlatform, ok := middleware2.GetForcePlatformFromContext(c); ok {
		platform = forcePlatform
	} else if apiKey.Group != nil {
		platform = apiKey.Group.Platform
	}
	isGeminiProtocol := h.isGeminiProtocolPlatform(c.Request.Context(), platform)
	sessionKey := sessionHash
	if isGeminiProtocol && sessionHash != "" {
		sessionKey = "gemini:" + sessionHash
	}

	// 查询粘性会话绑定的账号 ID
	var sessionBoundAccountID int64
	if sessionKey != "" {
		sessionBoundAccountID, _ = h.gatewayService.GetCachedSessionAccountID(c.Request.Context(), apiKey.GroupID, sessionKey)
		// [DEBUG-STICKY] 打印粘性会话查询结果
		reqLog.Info("sticky.cache_lookup",
			zap.String("session_key", sessionKey),
			zap.Int64("bound_account_id", sessionBoundAccountID),
		)
		if sessionBoundAccountID > 0 {
			prefetchedGroupID := int64(0)
			if apiKey.GroupID != nil {
				prefetchedGroupID = *apiKey.GroupID
			}
			ctx := service.WithPrefetchedStickySession(c.Request.Context(), sessionBoundAccountID, prefetchedGroupID, h.metadataBridgeEnabled())
			c.Request = c.Request.WithContext(ctx)
		}
	} else {
		reqLog.Info("sticky.no_session_key", zap.String("session_hash", sessionHash))
	}
	// 判断是否真的绑定了粘性会话：有 sessionKey 且已经绑定到某个账号
	hasBoundSession := sessionKey != "" && sessionBoundAccountID > 0

	if isGeminiProtocol {
		fs := NewFailoverState(h.maxAccountSwitchesGemini, hasBoundSession)

		// 单账号分组提前设置 SingleAccountRetry 标记，让 Service 层首次 503 就不设模型限流标记。
		// 避免单账号分组收到 503 (MODEL_CAPACITY_EXHAUSTED) 时设 29s 限流，导致后续请求连续快速失败。
		if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), apiKey.GroupID) {
			ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
			c.Request = c.Request.WithContext(ctx)
		}

		for {
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, sessionKey, reqModel, fs.FailedAccountIDs, "", int64(0)) // Gemini 不使用会话限制
			if err != nil {
				if len(fs.FailedAccountIDs) == 0 {
					cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, reqModel, reqModel, service.PlatformGemini)
					if !cls.ModelNotFound {
						markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
					}
					reqLog.Warn("gateway.select_account_no_available",
						zap.String("model", reqModel),
						zap.Int64p("group_id", apiKey.GroupID),
						zap.String("platform", platform),
						zap.Bool("model_not_found", cls.ModelNotFound),
						zap.Error(err),
					)
					message := cls.Message
					if !cls.ModelNotFound {
						message = "No available accounts: " + err.Error()
					}
					h.handleStreamingAwareError(c, cls.Status, cls.ErrType, message, streamStarted)
					return
				}
				action := fs.HandleSelectionExhausted(c.Request.Context())
				switch action {
				case FailoverContinue:
					ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
					c.Request = c.Request.WithContext(ctx)
					continue
				case FailoverCanceled:
					failoverClientGone(c)
					return
				default: // FailoverExhausted
					if fs.LastFailoverErr != nil {
						h.handleFailoverExhausted(c, fs.LastFailoverErr, platform, streamStarted)
					} else {
						h.handleFailoverExhaustedSimple(c, 502, streamStarted)
					}
					return
				}
			}
			account := selection.Account
			setOpsSelectedAccount(c, account.ID, account.Platform)

			// 检查请求拦截（预热请求、SUGGESTION MODE等）
			if account.IsInterceptWarmupEnabled() {
				interceptType := detectInterceptType(body, reqModel, parsedReq.MaxTokens, isClaudeCodeClient)
				if interceptType != InterceptTypeNone {
					if selection.Acquired && selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
					if reqStream {
						sendMockInterceptStream(c, reqModel, interceptType)
					} else {
						sendMockInterceptResponse(c, reqModel, interceptType)
					}
					return
				}
			}

			// 3. 获取账号并发槽位
			accountReleaseFunc := selection.ReleaseFunc
			if !selection.Acquired {
				if selection.WaitPlan == nil {
					markOpsRoutingCapacityLimited(c)
					reqLog.Warn("gateway.select_account_no_slot_no_wait_plan",
						zap.Int64("account_id", account.ID),
						zap.String("model", reqModel),
						zap.String("platform", platform),
					)
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
					return
				}
				accountWaitCounted := false
				canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
				if err != nil {
					reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				} else if !canWait {
					reqLog.Info("gateway.account_wait_queue_full",
						zap.Int64("account_id", account.ID),
						zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
					)
					h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
					return
				}
				if err == nil && canWait {
					accountWaitCounted = true
				}
				releaseWait := func() {
					if accountWaitCounted {
						h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
						accountWaitCounted = false
					}
				}

				accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
					c,
					account.ID,
					selection.WaitPlan.MaxConcurrency,
					selection.WaitPlan.Timeout,
					reqStream,
					&streamStarted,
				)
				if err != nil {
					reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
					releaseWait()
					h.handleConcurrencyError(c, err, "account", streamStarted)
					return
				}
				// Slot acquired: no longer waiting in queue.
				releaseWait()
				if err := h.gatewayService.BindStickySession(c.Request.Context(), apiKey.GroupID, sessionKey, account.ID); err != nil {
					reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			// 账号槽位/等待计数需要在超时或断开时安全回收
			accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

			// 转发请求 - 根据账号平台分流
			var result *service.ForwardResult
			requestCtx := c.Request.Context()
			if fs.SwitchCount > 0 {
				requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
			}
			// 记录 Forward 前已写入字节数，Forward 后若增加则说明 SSE 内容已发，禁止 failover
			writerSizeBeforeForward := c.Writer.Size()
			if account.Platform == service.PlatformAntigravity {
				result, err = h.antigravityGatewayService.ForwardGemini(
					requestCtx,
					c,
					account,
					reqModel,
					"generateContent",
					reqStream,
					body,
					hasBoundSession,
					service.WithForwardGeminiSession(derefGroupID(apiKey.GroupID), sessionKey),
				)
			} else {
				result, err = h.geminiCompatService.Forward(requestCtx, c, account, body)
			}
			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			if err != nil {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					// 流式内容已写入客户端，无法撤销，禁止 failover 以防止流拼接腐化
					if c.Writer.Size() != writerSizeBeforeForward {
						h.handleFailoverExhausted(c, failoverErr, platform, true)
						return
					}
					action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, account.GetPoolModeRetryCount(), failoverErr)
					switch action {
					case FailoverContinue:
						continue
					case FailoverExhausted:
						h.handleFailoverExhausted(c, fs.LastFailoverErr, platform, streamStarted)
						return
					case FailoverCanceled:
						failoverClientGone(c)
						return
					}
				}
				upstreamErrorAlreadyCommunicated := gatewayForwardErrorAlreadyCommunicated(c, writerSizeBeforeForward, err)
				wroteFallback := false
				if !upstreamErrorAlreadyCommunicated {
					wroteFallback = h.ensureForwardErrorResponse(c, streamStarted)
				}
				forwardFailedFields := []zap.Field{
					zap.Int64("account_id", account.ID),
					zap.String("account_name", account.Name),
					zap.String("account_platform", account.Platform),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Bool("upstream_error_response_already_written", upstreamErrorAlreadyCommunicated),
					zap.Error(err),
				}
				if account.Proxy != nil {
					forwardFailedFields = append(forwardFailedFields,
						zap.Int64("proxy_id", account.Proxy.ID),
						zap.String("proxy_name", account.Proxy.Name),
						zap.String("proxy_host", account.Proxy.Host),
						zap.Int("proxy_port", account.Proxy.Port),
					)
				} else if account.ProxyID != nil {
					forwardFailedFields = append(forwardFailedFields, zap.Int64p("proxy_id", account.ProxyID))
				}
				reqLog.Error("gateway.forward_failed", forwardFailedFields...)
				return
			}

			// RPM 计数递增（Forward 成功后）
			// 注意：TOCTOU 竞态是已知且可接受的设计权衡，与 WindowCost 一致的 soft-limit 模式。
			// 在高并发下可能短暂超出 RPM 限制，但不会导致请求失败。
			if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
				if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
					reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}

			// 捕获请求信息（用于异步记录，避免在 goroutine 中访问 gin.Context）
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			requestPayloadHash := service.HashUsageRequestPayload(body)
			inboundEndpoint := GetInboundEndpoint(c)
			upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)

			if result.ReasoningEffort == nil {
				result.ReasoningEffort = service.NormalizeClaudeOutputEffort(parsedReq.OutputEffort)
			}
			// 国产模型 thinking-enabled 默认 effort 填充：Kimi/GLM/MiniMax 这些不支持 effort 档位的
			// passback-required 上游，仅要 thinking 启用且 OutputEffort 未明确传递时，在 usage_log 写 "high"
			// 避免该字段长期为 NULL（详见 DefaultEffortForThinkingEnabled 文档）。
			if result.ReasoningEffort == nil && parsedReq.ThinkingEnabled {
				protocolModel := result.UpstreamModel
				if protocolModel == "" {
					protocolModel = result.Model
				}
				result.ReasoningEffort = service.DefaultEffortForThinkingEnabled(protocolModel)
			}

			// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
			// ForceCacheBilling 提前拍成标量，避免 worker 闭包保活 failover 状态里的响应体。
			forceCacheBilling := fs.ForceCacheBilling
			quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
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
					ForceCacheBilling:  forceCacheBilling,
					APIKeyService:      h.apiKeyService,
					ChannelUsageFields: channelMapping.ToUsageFields(reqModel, result.UpstreamModel),
				}); err != nil {
					logger.L().With(
						zap.String("component", "handler.gateway.messages"),
						zap.Int64("user_id", subject.UserID),
						zap.Int64("api_key_id", apiKey.ID),
						zap.Any("group_id", apiKey.GroupID),
						zap.String("model", reqModel),
						zap.Int64("account_id", account.ID),
					).Error("gateway.record_usage_failed", zap.Error(err))
				}
			})
			return
		}
	}

	currentAPIKey := apiKey
	currentSubscription := subscription
	var fallbackGroupID *int64
	if apiKey.Group != nil {
		fallbackGroupID = apiKey.Group.FallbackGroupIDOnInvalidRequest
	}
	fallbackUsed := false

	// 单账号分组提前设置 SingleAccountRetry 标记，让 Service 层首次 503 就不设模型限流标记。
	// 避免单账号分组收到 503 (MODEL_CAPACITY_EXHAUSTED) 时设 29s 限流，导致后续请求连续快速失败。
	if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), currentAPIKey.GroupID) {
		ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}

	for {
		fs := NewFailoverState(h.maxAccountSwitches, hasBoundSession)
		retryWithFallback := false

		for {
			attemptParsedReq, err := parsedReq.CloneForBody(body)
			if err != nil {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
				return
			}

			// 选择支持该模型的账号
			reqLog.Info("sticky.selecting_account",
				zap.String("session_key", sessionKey),
				zap.Int64("sticky_bound_account_id", sessionBoundAccountID),
				zap.Bool("has_bound_session", hasBoundSession),
				zap.Int("failed_account_count", len(fs.FailedAccountIDs)),
			)
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), currentAPIKey.GroupID, sessionKey, reqModel, fs.FailedAccountIDs, parsedReq.MetadataUserID, subject.UserID)
			if err != nil {
				if len(fs.FailedAccountIDs) == 0 {
					cls := classifyNoAccountErrorFromGin(c, h.gatewayService, currentAPIKey, reqModel, reqModel, platform)
					if !cls.ModelNotFound {
						markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
					}
					reqLog.Warn("gateway.select_account_no_available",
						zap.String("model", reqModel),
						zap.Int64p("group_id", currentAPIKey.GroupID),
						zap.String("platform", platform),
						zap.Bool("fallback_used", fallbackUsed),
						zap.Bool("model_not_found", cls.ModelNotFound),
						zap.Error(err),
					)
					message := cls.Message
					if !cls.ModelNotFound {
						message = "No available accounts: " + err.Error()
					}
					h.handleStreamingAwareError(c, cls.Status, cls.ErrType, message, streamStarted)
					return
				}
				action := fs.HandleSelectionExhausted(c.Request.Context())
				switch action {
				case FailoverContinue:
					ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
					c.Request = c.Request.WithContext(ctx)
					continue
				case FailoverCanceled:
					failoverClientGone(c)
					return
				default: // FailoverExhausted
					if fs.LastFailoverErr != nil {
						h.handleFailoverExhausted(c, fs.LastFailoverErr, platform, streamStarted)
					} else {
						h.handleFailoverExhaustedSimple(c, 502, streamStarted)
					}
					return
				}
			}
			account := selection.Account
			setOpsSelectedAccount(c, account.ID, account.Platform)

			// [DEBUG-STICKY] 打印账号选择结果
			reqLog.Info("sticky.account_selected",
				zap.Int64("selected_account_id", account.ID),
				zap.String("account_name", account.Name),
				zap.Bool("slot_acquired", selection.Acquired),
				zap.Bool("has_wait_plan", selection.WaitPlan != nil),
				zap.Int64("sticky_bound_account_id", sessionBoundAccountID),
				zap.Bool("sticky_honored", sessionBoundAccountID > 0 && sessionBoundAccountID == account.ID),
			)

			// 检查请求拦截（预热请求、SUGGESTION MODE等）
			if account.IsInterceptWarmupEnabled() {
				interceptType := detectInterceptType(body, reqModel, parsedReq.MaxTokens, isClaudeCodeClient)
				if interceptType != InterceptTypeNone {
					if selection.Acquired && selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
					if reqStream {
						sendMockInterceptStream(c, reqModel, interceptType)
					} else {
						sendMockInterceptResponse(c, reqModel, interceptType)
					}
					return
				}
			}

			// 3. 获取账号并发槽位
			accountReleaseFunc := selection.ReleaseFunc
			if !selection.Acquired {
				if selection.WaitPlan == nil {
					markOpsRoutingCapacityLimited(c)
					reqLog.Warn("gateway.select_account_no_slot_no_wait_plan",
						zap.Int64("account_id", account.ID),
						zap.String("model", reqModel),
						zap.String("platform", platform),
					)
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
					return
				}
				accountWaitCounted := false
				canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
				if err != nil {
					reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				} else if !canWait {
					reqLog.Info("gateway.account_wait_queue_full",
						zap.Int64("account_id", account.ID),
						zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
					)
					h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
					return
				}
				if err == nil && canWait {
					accountWaitCounted = true
				}
				releaseWait := func() {
					if accountWaitCounted {
						h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
						accountWaitCounted = false
					}
				}

				accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
					c,
					account.ID,
					selection.WaitPlan.MaxConcurrency,
					selection.WaitPlan.Timeout,
					reqStream,
					&streamStarted,
				)
				if err != nil {
					reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
					releaseWait()
					h.handleConcurrencyError(c, err, "account", streamStarted)
					return
				}
				// Slot acquired: no longer waiting in queue.
				releaseWait()
				reqLog.Info("sticky.bind_after_wait",
					zap.String("session_key", sessionKey),
					zap.Int64("account_id", account.ID),
				)
				if err := h.gatewayService.BindStickySession(c.Request.Context(), currentAPIKey.GroupID, sessionKey, account.ID); err != nil {
					reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			// 账号槽位/等待计数需要在超时或断开时安全回收
			accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

			// ===== 用户消息串行队列 START =====
			var queueRelease func()
			umqMode := h.getUserMsgQueueMode(account, attemptParsedReq)

			switch umqMode {
			case config.UMQModeSerialize:
				// 串行模式：获取锁 + RPM 延迟 + 释放（当前行为不变）
				baseRPM := account.GetBaseRPM()
				release, qErr := h.userMsgQueueHelper.AcquireWithWait(
					c, account.ID, baseRPM, reqStream, &streamStarted,
					h.cfg.Gateway.UserMessageQueue.WaitTimeout(),
					reqLog,
				)
				if qErr != nil {
					// fail-open: 记录 warn，不阻止请求
					reqLog.Warn("gateway.umq_acquire_failed",
						zap.Int64("account_id", account.ID),
						zap.Error(qErr),
					)
				} else {
					queueRelease = release
				}

			case config.UMQModeThrottle:
				// 软性限速：仅施加 RPM 自适应延迟，不阻塞并发
				baseRPM := account.GetBaseRPM()
				if tErr := h.userMsgQueueHelper.ThrottleWithPing(
					c, account.ID, baseRPM, reqStream, &streamStarted,
					h.cfg.Gateway.UserMessageQueue.WaitTimeout(),
					reqLog,
				); tErr != nil {
					reqLog.Warn("gateway.umq_throttle_failed",
						zap.Int64("account_id", account.ID),
						zap.Error(tErr),
					)
				}

			default:
				if umqMode != "" {
					reqLog.Warn("gateway.umq_unknown_mode",
						zap.String("mode", umqMode),
						zap.Int64("account_id", account.ID),
					)
				}
			}

			// 用 wrapReleaseOnDone 确保 context 取消时自动释放（仅 serialize 模式有 queueRelease）
			queueRelease = wrapReleaseOnDone(c.Request.Context(), queueRelease)
			// 注入回调到 ParsedRequest：使用外层 wrapper 以便提前清理 AfterFunc
			attemptParsedReq.OnUpstreamAccepted = queueRelease
			// ===== 用户消息串行队列 END =====

			// 渠道模型映射只作用于本次账号尝试，避免 failover 后污染原始 ParsedRequest。
			if channelMapping.Mapped {
				attemptParsedReq.Model = channelMapping.MappedModel
				if err := attemptParsedReq.ReplaceBody(h.gatewayService.ReplaceModelInBody(attemptParsedReq.Body.Bytes(), channelMapping.MappedModel)); err != nil {
					h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
					return
				}
			}
			// Bedrock CC 兼容：清理 body 专有字段 + 过滤 anthropic-beta header，适用于所有转发路径
			if err := attemptParsedReq.ReplaceBody(h.gatewayService.ApplyBedrockCCCompat(c, attemptParsedReq.Body.Bytes(), attemptParsedReq.Model, account, apiKey.GroupID)); err != nil {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
				return
			}
			attemptBody := attemptParsedReq.Body.Bytes()

			// 转发请求 - 根据账号平台分流
			c.Set("parsed_request", attemptParsedReq)
			var result *service.ForwardResult
			requestCtx := c.Request.Context()
			if fs.SwitchCount > 0 {
				requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
			}
			// 记录 Forward 前已写入字节数，Forward 后若增加则说明 SSE 内容已发，禁止 failover
			writerSizeBeforeForward := c.Writer.Size()
			if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
				result, err = h.antigravityGatewayService.Forward(requestCtx, c, account, attemptBody, hasBoundSession)
			} else {
				result, err = h.gatewayService.Forward(requestCtx, c, account, attemptParsedReq)
			}

			// 兜底释放串行锁（正常情况已通过回调提前释放）
			if queueRelease != nil {
				queueRelease()
			}
			// 清理回调引用，防止 failover 重试时旧回调被错误调用
			attemptParsedReq.OnUpstreamAccepted = nil

			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			if err != nil {
				// Beta policy block: return 400 immediately, no failover
				var betaBlockedErr *service.BetaBlockedError
				if errors.As(err, &betaBlockedErr) {
					service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
					h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", betaBlockedErr.Message)
					return
				}

				var promptTooLongErr *service.PromptTooLongError
				if errors.As(err, &promptTooLongErr) {
					reqLog.Warn("gateway.prompt_too_long_from_antigravity",
						zap.Any("current_group_id", currentAPIKey.GroupID),
						zap.Any("fallback_group_id", fallbackGroupID),
						zap.Bool("fallback_used", fallbackUsed),
					)
					if !fallbackUsed && fallbackGroupID != nil && *fallbackGroupID > 0 {
						fallbackGroup, err := h.gatewayService.ResolveGroupByID(c.Request.Context(), *fallbackGroupID)
						if err != nil {
							reqLog.Warn("gateway.resolve_fallback_group_failed", zap.Int64("fallback_group_id", *fallbackGroupID), zap.Error(err))
							_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
							return
						}
						if fallbackGroup.Platform != service.PlatformAnthropic ||
							fallbackGroup.SubscriptionType == service.SubscriptionTypeSubscription ||
							fallbackGroup.FallbackGroupIDOnInvalidRequest != nil {
							reqLog.Warn("gateway.fallback_group_invalid",
								zap.Int64("fallback_group_id", fallbackGroup.ID),
								zap.String("fallback_platform", fallbackGroup.Platform),
								zap.String("fallback_subscription_type", fallbackGroup.SubscriptionType),
							)
							_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
							return
						}
						fallbackAPIKey := cloneAPIKeyWithGroup(apiKey, fallbackGroup)
						if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), fallbackAPIKey.User, fallbackAPIKey, fallbackGroup, nil, service.PlatformFromAPIKey(fallbackAPIKey)); err != nil {
							status, code, message, retryAfter := billingErrorDetails(err)
							if retryAfter > 0 {
								c.Header("Retry-After", strconv.Itoa(retryAfter))
							}
							h.handleStreamingAwareError(c, status, code, message, streamStarted)
							return
						}
						// 兜底重试按"直接请求兜底分组"处理：清除强制平台，允许按分组平台调度
						ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, "")
						c.Request = c.Request.WithContext(ctx)
						currentAPIKey = fallbackAPIKey
						currentSubscription = nil
						fallbackUsed = true
						retryWithFallback = true
						break
					}
					_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
					return
				}
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					// 流式内容已写入客户端，无法撤销，禁止 failover 以防止流拼接腐化
					if c.Writer.Size() != writerSizeBeforeForward {
						h.handleFailoverExhausted(c, failoverErr, account.Platform, true)
						return
					}
					action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, account.GetPoolModeRetryCount(), failoverErr)
					switch action {
					case FailoverContinue:
						continue
					case FailoverExhausted:
						h.handleFailoverExhausted(c, fs.LastFailoverErr, account.Platform, streamStarted)
						return
					case FailoverCanceled:
						failoverClientGone(c)
						return
					}
				}
				upstreamErrorAlreadyCommunicated := gatewayForwardErrorAlreadyCommunicated(c, writerSizeBeforeForward, err)
				wroteFallback := false
				if !upstreamErrorAlreadyCommunicated {
					wroteFallback = h.ensureForwardErrorResponse(c, streamStarted)
				}
				forwardFailedFields := []zap.Field{
					zap.Int64("account_id", account.ID),
					zap.String("account_name", account.Name),
					zap.String("account_platform", account.Platform),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Bool("upstream_error_response_already_written", upstreamErrorAlreadyCommunicated),
					zap.Error(err),
				}
				if account.Proxy != nil {
					forwardFailedFields = append(forwardFailedFields,
						zap.Int64("proxy_id", account.Proxy.ID),
						zap.String("proxy_name", account.Proxy.Name),
						zap.String("proxy_host", account.Proxy.Host),
						zap.Int("proxy_port", account.Proxy.Port),
					)
				} else if account.ProxyID != nil {
					forwardFailedFields = append(forwardFailedFields, zap.Int64p("proxy_id", account.ProxyID))
				}
				reqLog.Error("gateway.forward_failed", forwardFailedFields...)
				return
			}

			// RPM 计数递增（Forward 成功后）
			// 注意：TOCTOU 竞态是已知且可接受的设计权衡，与 WindowCost 一致的 soft-limit 模式。
			// 在高并发下可能短暂超出 RPM 限制，但不会导致请求失败。
			if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
				if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
					reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}

			// 绑定粘性会话（成功转发后绑定/刷新）
			// - 无现有绑定（首次请求）：创建绑定
			// - 选中账号与粘性账号一致：刷新 TTL
			// - 粘性账号因负载/RPM 被跳过、选中了其他账号：不覆盖原绑定，
			//   下次请求粘性账号恢复后仍可命中
			if sessionKey != "" && (sessionBoundAccountID == 0 || sessionBoundAccountID == account.ID) {
				if err := h.gatewayService.BindStickySession(c.Request.Context(), currentAPIKey.GroupID, sessionKey, account.ID); err != nil {
					reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}

			// 捕获请求信息（用于异步记录，避免在 goroutine 中访问 gin.Context）
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			// Forward 内部可能继续改写 body，usage 去重指纹必须使用最终上游接受的当前 body。
			requestPayloadHash := service.HashUsageRequestPayload(attemptParsedReq.Body.Bytes())
			inboundEndpoint := GetInboundEndpoint(c)
			upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)

			if result.ReasoningEffort == nil {
				result.ReasoningEffort = service.NormalizeClaudeOutputEffort(attemptParsedReq.OutputEffort)
			}
			// 同上（重试路径中的对称填充）。详见非重试路径同名注释。
			if result.ReasoningEffort == nil && attemptParsedReq.ThinkingEnabled {
				protocolModel := result.UpstreamModel
				if protocolModel == "" {
					protocolModel = result.Model
				}
				result.ReasoningEffort = service.DefaultEffortForThinkingEnabled(protocolModel)
			}

			// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
			// ForceCacheBilling 提前拍成标量，避免 worker 闭包保活 failover 状态里的响应体。
			forceCacheBilling := fs.ForceCacheBilling
			quotaPlatform := service.QuotaPlatform(c.Request.Context(), currentAPIKey)
			h.submitUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
					Result:             result,
					QuotaPlatform:      quotaPlatform,
					APIKey:             currentAPIKey,
					User:               currentAPIKey.User,
					Account:            account,
					Subscription:       currentSubscription,
					InboundEndpoint:    inboundEndpoint,
					UpstreamEndpoint:   upstreamEndpoint,
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: requestPayloadHash,
					ForceCacheBilling:  forceCacheBilling,
					APIKeyService:      h.apiKeyService,
					ChannelUsageFields: channelMapping.ToUsageFields(reqModel, result.UpstreamModel),
				}); err != nil {
					logger.L().With(
						zap.String("component", "handler.gateway.messages"),
						zap.Int64("user_id", subject.UserID),
						zap.Int64("api_key_id", currentAPIKey.ID),
						zap.Any("group_id", currentAPIKey.GroupID),
						zap.String("model", reqModel),
						zap.Int64("account_id", account.ID),
					).Error("gateway.record_usage_failed", zap.Error(err))
				}
			})
			return
		}
		if !retryWithFallback {
			return
		}
	}
}

// Models handles listing available models
// GET /v1/models
// Returns models based on account configurations (model_mapping whitelist)
// Falls back to default models if no whitelist is configured
func (h *GatewayHandler) Models(c *gin.Context) {
	apiKey, _ := middleware2.GetAPIKeyFromContext(c)

	var groupID *int64
	var platform string

	if apiKey != nil && apiKey.Group != nil {
		groupID = &apiKey.Group.ID
		platform = apiKey.Group.Platform
	}
	if forcedPlatform, ok := middleware2.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forcedPlatform) != "" {
		platform = forcedPlatform
	}

	includeEntryProtocols := c.Query("include_entry_protocols") == "1" || strings.EqualFold(c.Query("include_entry_protocols"), "true")
	pricedDetails := h.gatewayService.GetPricedModelDetails(c.Request.Context(), groupID, platform)
	availableModels := modelNamesFromPricedDetails(pricedDetails)
	if service.NormalizePlatformSlug(platform) == service.PlatformAnthropic {
		availableModels = mergeModelIDs(availableModels, h.gatewayService.GetAvailableModels(c.Request.Context(), groupID, platform))
	}
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.CustomModelsListEnabled() {
		if len(apiKey.Group.ModelsListConfig.Models) == 0 {
			if includeEntryProtocols {
				writeModelsListWithEntryProtocolMetadata(c, h.gatewayService, h.platformService, apiKey, nil)
				return
			}
			writeCustomModelsList(c, platform, nil)
			return
		}
		fallbackModels := customModelsListFallbackModels(platform)
		availableModels = filterModelsByCustomList(customModelsListSource(platform, availableModels, fallbackModels), fallbackModels, apiKey.Group.ModelsListConfig.Models)
		if includeEntryProtocols {
			writeModelsListWithEntryProtocolMetadata(c, h.gatewayService, h.platformService, apiKey, filterPricedModelDetailsByIDs(pricedDetails, availableModels))
			return
		}
		writeCustomModelsList(c, platform, availableModels)
		return
	}

	if includeEntryProtocols {
		writeModelsListWithEntryProtocolMetadata(c, h.gatewayService, h.platformService, apiKey, filterPricedModelDetailsByIDs(pricedDetails, availableModels))
		return
	}
	writeCustomModelsList(c, platform, availableModels)
}

type modelListEntryProtocolItem struct {
	ID     string                       `json:"id"`
	Object string                       `json:"object"`
	XimoAI modelListEntryProtocolXimoAI `json:"ximoai"`
}

type modelListEntryProtocolXimoAI struct {
	DefaultEntryProtocol string                       `json:"default_entry_protocol"`
	DefaultEndpoint      string                       `json:"default_endpoint"`
	DefaultEntryID       string                       `json:"default_entry_id,omitempty"`
	Method               string                       `json:"method"`
	RequestContentType   string                       `json:"request_content_type"`
	StreamEndpoint       string                       `json:"stream_endpoint,omitempty"`
	ModelType            string                       `json:"model_type"`
	OperationType        string                       `json:"operation_type"`
	ExecutionMode        string                       `json:"execution_mode"`
	SupportsStream       bool                         `json:"supports_stream"`
	SupportsPolling      bool                         `json:"supports_polling"`
	RequestContract      map[string]any               `json:"request_contract,omitempty"`
	ResponseContract     map[string]any               `json:"response_contract,omitempty"`
	StreamContract       map[string]any               `json:"stream_contract,omitempty"`
	ToolContract         map[string]any               `json:"tool_contract,omitempty"`
	ThinkingContract     map[string]any               `json:"thinking_contract,omitempty"`
	MediaContract        map[string]any               `json:"media_contract,omitempty"`
	PollingContract      map[string]any               `json:"polling_contract,omitempty"`
	UnsupportedCaps      []string                     `json:"unsupported_capabilities,omitempty"`
	EntryProtocols       []modelListEntryProtocolInfo `json:"entry_protocols,omitempty"`
	Group                *modelListEntryProtocolGroup `json:"group,omitempty"`
	Pricing              *userSupportedModelPricing   `json:"pricing"`
}

type modelListEntryProtocolInfo struct {
	ID                 string         `json:"id"`
	Protocol           string         `json:"protocol"`
	Endpoint           string         `json:"endpoint"`
	Method             string         `json:"method"`
	RequestContentType string         `json:"request_content_type"`
	StreamEndpoint     string         `json:"stream_endpoint,omitempty"`
	SupportsStream     bool           `json:"supports_stream"`
	RequestContract    map[string]any `json:"request_contract,omitempty"`
	ResponseContract   map[string]any `json:"response_contract,omitempty"`
	StreamContract     map[string]any `json:"stream_contract,omitempty"`
	ToolContract       map[string]any `json:"tool_contract,omitempty"`
	ThinkingContract   map[string]any `json:"thinking_contract,omitempty"`
	MediaContract      map[string]any `json:"media_contract,omitempty"`
	PollingContract    map[string]any `json:"polling_contract,omitempty"`
}

type modelListEntryProtocolGroup struct {
	ID                      int64   `json:"id"`
	Name                    string  `json:"name"`
	SubscriptionType        string  `json:"subscription_type"`
	IsExclusive             bool    `json:"is_exclusive"`
	RateMultiplier          float64 `json:"rate_multiplier"`
	EffectiveRateMultiplier float64 `json:"effective_rate_multiplier"`
}

func writeModelsListWithEntryProtocolMetadata(c *gin.Context, gatewaySvc *service.GatewayService, platformSvc *service.PlatformService, apiKey *service.APIKey, details []service.GatewayPricedModelDetail) {
	models := make([]modelListEntryProtocolItem, 0, len(details))
	group := modelListEntryProtocolGroupFromAPIKey(c.Request.Context(), gatewaySvc, apiKey)
	for _, detail := range details {
		meta := publicEntryMetadataForPricedModel(detail, platformForEntryMetadata(c.Request.Context(), platformSvc, detail.Platform))
		models = append(models, modelListEntryProtocolItem{
			ID:     detail.Name,
			Object: "model",
			XimoAI: modelListEntryProtocolXimoAI{
				DefaultEntryProtocol: meta.DefaultEntryProtocol,
				DefaultEndpoint:      meta.DefaultEndpoint,
				DefaultEntryID:       meta.DefaultEntryID,
				Method:               meta.Method,
				RequestContentType:   meta.RequestContentType,
				StreamEndpoint:       meta.StreamEndpoint,
				ModelType:            meta.ModelType,
				OperationType:        meta.OperationType,
				ExecutionMode:        meta.ExecutionMode,
				SupportsStream:       meta.SupportsStream,
				SupportsPolling:      meta.SupportsPolling,
				RequestContract:      meta.RequestContract,
				ResponseContract:     meta.ResponseContract,
				StreamContract:       meta.StreamContract,
				ToolContract:         meta.ToolContract,
				ThinkingContract:     meta.ThinkingContract,
				MediaContract:        meta.MediaContract,
				PollingContract:      meta.PollingContract,
				UnsupportedCaps:      meta.UnsupportedCaps,
				EntryProtocols:       meta.EntryProtocols,
				Group:                group,
				Pricing:              toUserPricing(scalePricingForModelList(detail.Pricing, group)),
			},
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

func modelListEntryProtocolGroupFromAPIKey(ctx context.Context, gatewaySvc *service.GatewayService, apiKey *service.APIKey) *modelListEntryProtocolGroup {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	group := apiKey.Group
	groupID := group.ID
	if apiKey.GroupID != nil && *apiKey.GroupID > 0 {
		groupID = *apiKey.GroupID
	}
	effectiveRate := group.RateMultiplier
	if gatewaySvc != nil && apiKey.UserID > 0 && groupID > 0 {
		effectiveRate = gatewaySvc.ResolveEffectiveGroupRateMultiplier(ctx, apiKey.UserID, groupID, group.RateMultiplier)
	}
	return &modelListEntryProtocolGroup{
		ID:                      groupID,
		Name:                    group.Name,
		SubscriptionType:        group.SubscriptionType,
		IsExclusive:             group.IsExclusive,
		RateMultiplier:          group.RateMultiplier,
		EffectiveRateMultiplier: effectiveRate,
	}
}

func scalePricingForModelList(p *service.ChannelModelPricing, group *modelListEntryProtocolGroup) *service.ChannelModelPricing {
	if p == nil {
		return nil
	}
	rate := 1.0
	if group != nil && group.EffectiveRateMultiplier > 0 {
		rate = group.EffectiveRateMultiplier
	}
	if rate == 1 {
		cp := p.Clone()
		return &cp
	}

	cp := p.Clone()
	scaleFloatPtr := func(v **float64) {
		if v == nil || *v == nil {
			return
		}
		scaled := **v * rate
		*v = &scaled
	}

	scaleFloatPtr(&cp.InputPrice)
	scaleFloatPtr(&cp.OutputPrice)
	scaleFloatPtr(&cp.CacheWritePrice)
	scaleFloatPtr(&cp.CacheReadPrice)
	scaleFloatPtr(&cp.ImageOutputPrice)
	scaleFloatPtr(&cp.PerRequestPrice)
	for i := range cp.Intervals {
		scaleFloatPtr(&cp.Intervals[i].InputPrice)
		scaleFloatPtr(&cp.Intervals[i].OutputPrice)
		scaleFloatPtr(&cp.Intervals[i].CacheWritePrice)
		scaleFloatPtr(&cp.Intervals[i].CacheReadPrice)
		scaleFloatPtr(&cp.Intervals[i].PerRequestPrice)
	}
	return &cp
}

type publicEntryMetadata struct {
	DefaultEntryProtocol string
	DefaultEndpoint      string
	DefaultEntryID       string
	Method               string
	RequestContentType   string
	StreamEndpoint       string
	ModelType            string
	OperationType        string
	ExecutionMode        string
	SupportsStream       bool
	SupportsPolling      bool
	RequestContract      map[string]any
	ResponseContract     map[string]any
	StreamContract       map[string]any
	ToolContract         map[string]any
	ThinkingContract     map[string]any
	MediaContract        map[string]any
	PollingContract      map[string]any
	UnsupportedCaps      []string
	EntryProtocols       []modelListEntryProtocolInfo
}

func platformForEntryMetadata(ctx context.Context, platformSvc *service.PlatformService, slug string) *service.Platform {
	if strings.TrimSpace(slug) == "" {
		return nil
	}
	if platformSvc == nil {
		platformSvc = service.NewPlatformService(nil)
	}
	platform, err := platformSvc.GetBySlug(ctx, slug)
	if err != nil {
		return nil
	}
	return platform
}

func publicEntryMetadataForPricedModel(detail service.GatewayPricedModelDetail, platformInfo *service.Platform) publicEntryMetadata {
	protocol, endpoint := defaultEntryProtocolForPricedModel(detail, platformInfo)
	modelType := publicModelTypeForPricedModel(detail, endpoint)
	operationType := publicOperationTypeForPricedModel(detail, endpoint, modelType)
	executionMode := "sync"
	supportsPolling := false
	if modelType == "video" {
		executionMode = "async"
		supportsPolling = true
	}
	entryID := publicEntryID(protocol, endpoint, operationType)
	method := publicMethodForEntry(operationType)
	requestContentType := publicRequestContentTypeForEntry(operationType)
	streamEndpoint := publicStreamEndpointForEntry(protocol, endpoint, modelType)
	supportsStream := streamEndpoint != ""
	requestContract := publicRequestContractForPricedModel(detail, protocol, endpoint, modelType, operationType)
	responseContract := publicResponseContractForPricedModel(detail, protocol, endpoint, modelType, operationType)
	streamContract := publicStreamContractForPricedModel(protocol, endpoint, streamEndpoint, modelType, operationType)
	toolContract := publicToolContractForPricedModel(protocol, endpoint, modelType, operationType)
	thinkingContract := publicThinkingContractForPricedModel(protocol, endpoint, modelType, operationType)
	mediaContract := publicMediaContractForPricedModel(protocol, endpoint, modelType, operationType)
	pollingContract := publicPollingContractForPricedModel(endpoint, operationType)
	unsupportedCaps := publicUnsupportedCapabilitiesForPricedModel(protocol, endpoint, modelType, operationType)
	return publicEntryMetadata{
		DefaultEntryProtocol: protocol,
		DefaultEndpoint:      endpoint,
		DefaultEntryID:       entryID,
		Method:               method,
		RequestContentType:   requestContentType,
		StreamEndpoint:       streamEndpoint,
		ModelType:            modelType,
		OperationType:        operationType,
		ExecutionMode:        executionMode,
		SupportsStream:       supportsStream,
		SupportsPolling:      supportsPolling,
		RequestContract:      requestContract,
		ResponseContract:     responseContract,
		StreamContract:       streamContract,
		ToolContract:         toolContract,
		ThinkingContract:     thinkingContract,
		MediaContract:        mediaContract,
		PollingContract:      pollingContract,
		UnsupportedCaps:      unsupportedCaps,
		EntryProtocols: []modelListEntryProtocolInfo{{
			ID:                 entryID,
			Protocol:           protocol,
			Endpoint:           endpoint,
			Method:             method,
			RequestContentType: requestContentType,
			StreamEndpoint:     streamEndpoint,
			SupportsStream:     supportsStream,
			RequestContract:    requestContract,
			ResponseContract:   responseContract,
			StreamContract:     streamContract,
			ToolContract:       toolContract,
			ThinkingContract:   thinkingContract,
			MediaContract:      mediaContract,
			PollingContract:    pollingContract,
		}},
	}
}

func defaultEntryProtocolForPricedModel(detail service.GatewayPricedModelDetail, platformInfo *service.Platform) (string, string) {
	model := strings.TrimSpace(detail.Name)
	platform := service.NormalizePlatformSlug(detail.Platform)
	mode := service.BillingModeToken
	if detail.Pricing != nil && detail.Pricing.BillingMode != "" {
		mode = detail.Pricing.BillingMode
	}

	switch platform {
	case service.PlatformOpenAI:
		return openAIEntryForPricedModel(detail, mode)
	case service.PlatformAnthropic:
		return "anthropic", "/v1/messages"
	case service.PlatformGemini:
		if mode == service.BillingModeVideo {
			return "gemini", "/v1beta/models/" + model + ":generateVideos"
		}
		return "gemini", "/v1beta/models/" + model + ":generateContent"
	case service.PlatformAntigravity:
		if strings.HasPrefix(strings.ToLower(model), "gemini") {
			return "gemini", "/antigravity/v1beta/models/" + model + ":generateContent"
		}
		return "anthropic", "/antigravity/v1/messages"
	case service.PlatformKlingAudio:
		return "openai", "/v1/audio/speech"
	case service.PlatformGrok:
		if mode == service.BillingModeVideo {
			return "openai", "/v1/videos"
		}
		return "openai", "/v1/videos"
	default:
		if platformInfo != nil {
			switch {
			case platformInfo.IsAnthropicCompatible():
				return "anthropic", "/v1/messages"
			case platformInfo.IsGeminiCompatible():
				if mode == service.BillingModeVideo {
					return "gemini", "/v1beta/models/" + model + ":generateVideos"
				}
				return "gemini", "/v1beta/models/" + model + ":generateContent"
			case platformInfo.IsOpenAICompatible():
				return openAIEntryForPricedModel(detail, mode)
			}
		}
		return openAIEntryForPricedModel(detail, mode)
	}
}

func openAIEntryForPricedModel(detail service.GatewayPricedModelDetail, mode service.BillingMode) (string, string) {
	if shouldUseOpenAIImageEndpoint(detail, mode) || mode == service.BillingModeImage {
		return "openai", "/v1/images/generations"
	}
	if isOpenAIAudioConversationModel(detail.Name, mode) {
		return "openai", "/v1/chat/completions"
	}
	if audioEndpoint := openAIAudioEndpointForModel(detail.Name, mode); audioEndpoint != "" {
		return "openai", audioEndpoint
	}
	if mode == service.BillingModeVideo {
		return "openai", "/v1/videos"
	}
	return "openai", "/v1/responses"
}

func shouldUseOpenAIImageEndpoint(detail service.GatewayPricedModelDetail, mode service.BillingMode) bool {
	if mode != service.BillingModePerRequest {
		return false
	}
	if detail.Pricing != nil && detail.Pricing.ImageOutputPrice != nil {
		return true
	}
	model := strings.ToLower(strings.TrimSpace(detail.Name))
	return strings.Contains(model, "image")
}

func openAIAudioEndpointForModel(model string, mode service.BillingMode) string {
	if mode == service.BillingModeImage || mode == service.BillingModeVideo {
		return ""
	}
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(model, "translat"):
		return "/v1/audio/translations"
	case strings.Contains(model, "transcrib") || strings.Contains(model, "transcript") || strings.Contains(model, "whisper") || strings.Contains(model, "stt"):
		return "/v1/audio/transcriptions"
	case strings.Contains(model, "tts") || strings.Contains(model, "speech"):
		return "/v1/audio/speech"
	default:
		return ""
	}
}

func isOpenAIAudioConversationModel(model string, mode service.BillingMode) bool {
	if mode == service.BillingModeImage || mode == service.BillingModeVideo {
		return false
	}
	model = strings.ToLower(strings.TrimSpace(model))
	if !strings.Contains(model, "audio") {
		return false
	}
	return openAIAudioEndpointForModel(model, mode) == ""
}

func publicModelTypeForPricedModel(detail service.GatewayPricedModelDetail, endpoint string) string {
	mode := service.BillingModeToken
	if detail.Pricing != nil && detail.Pricing.BillingMode != "" {
		mode = detail.Pricing.BillingMode
	}
	switch {
	case strings.Contains(endpoint, "/audio/transcriptions"):
		return "transcription"
	case strings.Contains(endpoint, "/audio/translations"):
		return "translation"
	case strings.Contains(endpoint, "/audio/speech"):
		return "audio"
	case isOpenAIAudioConversationModel(detail.Name, mode):
		return "audio"
	case strings.Contains(endpoint, "/videos") || strings.Contains(endpoint, ":generateVideos") || mode == service.BillingModeVideo:
		return "video"
	case strings.Contains(endpoint, "/images/") || mode == service.BillingModeImage || shouldUseOpenAIImageEndpoint(detail, mode):
		return "image"
	default:
		return "chat"
	}
}

func publicOperationTypeForPricedModel(detail service.GatewayPricedModelDetail, endpoint, modelType string) string {
	platform := service.NormalizePlatformSlug(detail.Platform)
	model := strings.ToLower(strings.TrimSpace(detail.Name))
	if platform == service.PlatformKlingAudio {
		switch model {
		case "kling-custom-voices":
			return "voice_management"
		case "kling-presets-voices":
			return "voice_catalog"
		default:
			return "audio_tts"
		}
	}
	switch modelType {
	case "audio":
		if strings.Contains(endpoint, "/v1/chat/completions") || isOpenAIAudioConversationModel(detail.Name, service.BillingModeToken) {
			return "chat_audio"
		}
		return "audio_tts"
	case "transcription":
		return "audio_transcription"
	case "translation":
		return "audio_translation"
	case "image":
		return "image_generation"
	case "video":
		return "video_generation"
	default:
		return "chat"
	}
}

func publicEntryID(protocol, endpoint, operationType string) string {
	switch {
	case protocol == "openai" && strings.Contains(endpoint, "/v1/responses"):
		return "openai_responses"
	case protocol == "openai" && strings.Contains(endpoint, "/v1/chat/completions"):
		return "openai_chat_completions"
	case protocol == "openai" && strings.Contains(endpoint, "/v1/images/generations"):
		return "openai_images"
	case protocol == "openai" && strings.Contains(endpoint, "/v1/audio/"):
		return "openai_audio"
	case protocol == "anthropic":
		return "anthropic_messages"
	case protocol == "gemini" && operationType == "video_generation":
		return "gemini_video"
	case protocol == "gemini":
		return "gemini_native"
	default:
		return protocol + "_" + operationType
	}
}

func publicMethodForEntry(operationType string) string {
	switch operationType {
	default:
		return http.MethodPost
	}
}

func publicRequestContentTypeForEntry(operationType string) string {
	switch operationType {
	case "audio_transcription", "audio_translation":
		return "multipart/form-data"
	default:
		return "application/json"
	}
}

func publicStreamEndpointForEntry(protocol, endpoint, modelType string) string {
	if modelType != "chat" {
		return ""
	}
	switch protocol {
	case "openai":
		if strings.Contains(endpoint, "/v1/responses") {
			return endpoint
		}
	case "anthropic":
		return endpoint
	case "gemini":
		return strings.Replace(endpoint, ":generateContent", ":streamGenerateContent?alt=sse", 1)
	}
	return ""
}

func publicRequestContractForPricedModel(detail service.GatewayPricedModelDetail, protocol, endpoint, modelType, operationType string) map[string]any {
	platform := service.NormalizePlatformSlug(detail.Platform)
	if modelType == "chat" {
		switch protocol {
		case "openai":
			if strings.Contains(endpoint, "/v1/responses") {
				return publicOpenAIResponsesChatRequestContract()
			}
		case "anthropic":
			return publicAnthropicMessagesChatRequestContract()
		case "gemini":
			return publicGeminiNativeChatRequestContract()
		}
	}
	if operationType == "image_generation" {
		switch protocol {
		case "gemini":
			return publicGeminiImageRequestContract()
		default:
			return publicOpenAIImageRequestContract()
		}
	}
	switch operationType {
	case "chat_audio":
		return map[string]any{
			"required_fields": []string{"model", "messages", "modalities", "audio"},
			"optional_fields": []string{},
			"fields": map[string]any{
				"model":      map[string]any{"type": "string", "location": "body.model"},
				"messages":   map[string]any{"type": "array", "location": "body.messages"},
				"modalities": map[string]any{"type": "array", "location": "body.modalities", "values": []string{"text", "audio"}},
				"audio": map[string]any{
					"type":     "object",
					"location": "body.audio",
					"fields": map[string]any{
						"voice":  map[string]any{"type": "string", "verified_values": []string{"alloy"}},
						"format": map[string]any{"type": "string", "verified_values": []string{"wav"}},
					},
				},
			},
			"examples": map[string]any{
				"modalities": []string{"text", "audio"},
				"audio": map[string]any{
					"voice":  "alloy",
					"format": "wav",
				},
			},
		}
	case "audio_tts":
		if platform == service.PlatformKlingAudio {
			return map[string]any{
				"required_fields": []string{"model", "input", "voice_id"},
				"optional_fields": []string{"voice_language", "voice_speed"},
				"field_notes": map[string]any{
					"voice_id": "Must be a Kling voice id, not OpenAI voice names such as alloy or nova.",
				},
				"examples": map[string]any{
					"voice_id":       "genshin_vindi2",
					"voice_language": "zh",
					"voice_speed":    1,
				},
			}
		}
		return map[string]any{
			"required_fields": []string{"model", "input", "voice"},
			"optional_fields": []string{"response_format", "speed"},
			"fields": map[string]any{
				"model":           map[string]any{"type": "string", "location": "body.model"},
				"input":           map[string]any{"type": "string", "location": "body.input"},
				"voice":           map[string]any{"type": "string", "location": "body.voice", "verified_values": []string{"alloy"}},
				"response_format": map[string]any{"type": "string", "location": "body.response_format", "verified_values": []string{"wav"}},
				"speed":           map[string]any{"type": "number", "location": "body.speed"},
			},
			"examples": map[string]any{
				"audio": map[string]any{
					"input":           "contract tts ok",
					"voice":           "alloy",
					"response_format": "wav",
				},
			},
		}
	case "voice_management":
		return map[string]any{
			"create_required_fields": []string{"model", "voice_name", "voice_url"},
			"query_required_fields":  []string{"model", "voice_id"},
			"field_notes": map[string]any{
				"voice_url": "A public voice reference file URL accepted by the upstream provider.",
			},
		}
	case "voice_catalog":
		return map[string]any{
			"optional_fields": []string{"pageNum", "pageSize", "page_num", "page_size"},
		}
	case "audio_transcription", "audio_translation":
		return map[string]any{
			"required_fields": []string{"model", "file"},
			"optional_fields": []string{"language", "prompt", "response_format", "temperature"},
		}
	case "video_generation":
		return map[string]any{
			"required_fields": []string{"model", "prompt"},
			"optional_fields": []string{"images", "image_urls", "reference_images", "aspect_ratio", "size", "duration_seconds"},
		}
	case "image_generation":
		return publicOpenAIImageRequestContract()
	default:
		return map[string]any{
			"required_fields": []string{"model", "messages"},
			"optional_fields": []string{"stream", "tools", "tool_choice", "response_format"},
		}
	}
}

func publicOpenAIResponsesChatRequestContract() map[string]any {
	return map[string]any{
		"required_fields": []string{"model", "input"},
		"optional_fields": []string{"stream", "tools", "tool_choice", "text", "reasoning", "previous_response_id", "store", "include", "max_output_tokens", "temperature", "top_p"},
		"fields": map[string]any{
			"model": map[string]any{
				"type":     "string",
				"location": "body.model",
			},
			"input": map[string]any{
				"type":     []string{"string", "array"},
				"location": "body.input",
			},
			"stream": map[string]any{
				"type":     "boolean",
				"location": "body.stream",
				"default":  false,
			},
			"tools": map[string]any{
				"type":            "array",
				"location":        "body.tools",
				"supported_types": []string{"function"},
				"unsupported_types": []string{
					"image_generation",
				},
			},
			"tool_choice": map[string]any{
				"type":     []string{"string", "object"},
				"location": "body.tool_choice",
			},
			"reasoning": map[string]any{
				"type":     "object",
				"location": "body.reasoning",
				"notes":    "parameter is accepted, but visible reasoning content is not exposed by the verified upstream response",
			},
		},
		"examples": map[string]any{
			"text": map[string]any{
				"input": "Reply exactly: ok",
				"store": false,
			},
			"function_tool": map[string]any{
				"tools": []map[string]any{{
					"type":        "function",
					"name":        "get_test_value",
					"description": "Return a test value",
					"parameters": map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"code": map[string]any{"type": "string"}},
						"required":             []string{"code"},
						"additionalProperties": false,
					},
					"strict": true,
				}},
				"tool_choice": map[string]any{"type": "function", "name": "get_test_value"},
			},
		},
	}
}

func publicAnthropicMessagesChatRequestContract() map[string]any {
	return map[string]any{
		"required_fields": []string{"model", "messages", "max_tokens"},
		"optional_fields": []string{"stream", "system", "tools", "tool_choice", "thinking", "temperature", "top_p", "metadata", "stop_sequences"},
		"fields": map[string]any{
			"model":      map[string]any{"type": "string", "location": "body.model"},
			"messages":   map[string]any{"type": "array", "location": "body.messages"},
			"max_tokens": map[string]any{"type": "integer", "location": "body.max_tokens"},
			"stream":     map[string]any{"type": "boolean", "location": "body.stream", "default": false},
			"tools": map[string]any{
				"type":     "array",
				"location": "body.tools",
				"schema":   "Anthropic tools use name, description, input_schema",
			},
			"thinking": map[string]any{
				"type":     "object",
				"location": "body.thinking",
				"example":  map[string]any{"type": "enabled", "budget_tokens": 512},
			},
		},
		"examples": map[string]any{
			"text": map[string]any{
				"max_tokens": 80,
				"messages":   []map[string]any{{"role": "user", "content": "Reply exactly: ok"}},
			},
		},
	}
}

func publicGeminiNativeChatRequestContract() map[string]any {
	return map[string]any{
		"required_fields": []string{"contents"},
		"optional_fields": []string{"systemInstruction", "generationConfig", "safetySettings", "tools", "toolConfig"},
		"fields": map[string]any{
			"contents":          map[string]any{"type": "array", "location": "body.contents"},
			"systemInstruction": map[string]any{"type": "object", "location": "body.systemInstruction"},
			"generationConfig":  map[string]any{"type": "object", "location": "body.generationConfig"},
			"tools": map[string]any{
				"type":     "array",
				"location": "body.tools",
				"schema":   "Gemini native function declarations use tools[].functionDeclarations[]",
			},
			"toolConfig": map[string]any{"type": "object", "location": "body.toolConfig"},
		},
		"examples": map[string]any{
			"text": map[string]any{
				"contents": []map[string]any{{
					"role":  "user",
					"parts": []map[string]any{{"text": "Reply exactly: ok"}},
				}},
			},
		},
	}
}

func publicOpenAIImageRequestContract() map[string]any {
	sizeContract := map[string]any{
		"type":   "enum",
		"values": []string{"1024x1024", "1536x1024", "1024x1536"},
		"aliases": map[string]any{
			"square":           "1024x1024",
			"1:1":              "1024x1024",
			"landscape":        "1536x1024",
			"wide":             "1536x1024",
			"portrait":         "1024x1536",
			"9:16":             "1024x1536",
			"mobile_wallpaper": "1024x1536",
		},
	}
	return map[string]any{
		"required_fields": []string{"model", "prompt"},
		"optional_fields": []string{"n", "size", "quality", "response_format", "background"},
		"fields": map[string]any{
			"model":  map[string]any{"type": "string", "location": "body.model"},
			"prompt": map[string]any{"type": "string", "location": "body.prompt"},
			"n":      map[string]any{"type": "integer", "location": "body.n", "default": 1},
			"size":   sizeContract,
			"quality": map[string]any{
				"type":     "string",
				"location": "body.quality",
			},
			"background": map[string]any{
				"type":     "string",
				"location": "body.background",
			},
			"response_format": map[string]any{
				"type":    "enum",
				"values":  []string{"b64_json"},
				"default": "b64_json",
				"notes":   "url timed out in live verification and is intentionally not advertised",
			},
		},
		"size": sizeContract,
		"examples": map[string]any{
			"image": map[string]any{
				"prompt":          "A simple verification icon",
				"n":               1,
				"size":            "1024x1024",
				"response_format": "b64_json",
			},
		},
	}
}

func publicGeminiImageRequestContract() map[string]any {
	return map[string]any{
		"required_fields": []string{"contents"},
		"optional_fields": []string{"generationConfig", "safetySettings"},
		"fields": map[string]any{
			"contents":         map[string]any{"type": "array", "location": "body.contents"},
			"generationConfig": map[string]any{"type": "object", "location": "body.generationConfig"},
			"safetySettings":   map[string]any{"type": "array", "location": "body.safetySettings"},
		},
		"generationConfig": map[string]any{
			"responseModalities": map[string]any{
				"type":    "array",
				"values":  []string{"TEXT", "IMAGE"},
				"default": []string{"TEXT", "IMAGE"},
			},
			"imageConfig": map[string]any{
				"aspectRatio": map[string]any{
					"type":   "enum",
					"values": []string{"1:1", "16:9", "9:16", "4:3", "3:4"},
					"aliases": map[string]any{
						"square":           "1:1",
						"landscape":        "16:9",
						"wide":             "16:9",
						"portrait":         "9:16",
						"mobile_wallpaper": "9:16",
					},
				},
				"imageSize": map[string]any{
					"type":   "enum",
					"values": []string{"1K", "2K", "4K"},
					"aliases": map[string]any{
						"standard":        "1K",
						"hd":              "2K",
						"high_definition": "2K",
						"2k":              "2K",
						"4k":              "4K",
						"ultra_hd":        "4K",
					},
				},
			},
		},
		"examples": map[string]any{
			"image": map[string]any{
				"contents": []map[string]any{{
					"role":  "user",
					"parts": []map[string]any{{"text": "A simple verification icon"}},
				}},
				"generationConfig": map[string]any{
					"responseModalities": []string{"TEXT", "IMAGE"},
					"imageConfig":        map[string]any{"aspectRatio": "1:1", "imageSize": "1K"},
				},
			},
		},
	}
}

func publicResponseContractForPricedModel(detail service.GatewayPricedModelDetail, protocol, endpoint, modelType, operationType string) map[string]any {
	platform := service.NormalizePlatformSlug(detail.Platform)
	if modelType == "chat" {
		switch protocol {
		case "openai":
			if strings.Contains(endpoint, "/v1/responses") {
				return publicOpenAIResponsesChatResponseContract()
			}
		case "anthropic":
			return publicAnthropicMessagesChatResponseContract()
		case "gemini":
			return publicGeminiNativeChatResponseContract()
		}
	}
	if operationType == "image_generation" {
		switch protocol {
		case "gemini":
			return publicGeminiImageResponseContract()
		default:
			return publicOpenAIImageResponseContract()
		}
	}
	switch operationType {
	case "chat_audio":
		return map[string]any{
			"delivery":           "openai_chat_audio_base64",
			"audio_data_path":    "choices[0].message.audio.data",
			"transcript_path":    "choices[0].message.audio.transcript",
			"audio_id_path":      "choices[0].message.audio.id",
			"expires_at_path":    "choices[0].message.audio.expires_at",
			"usage_path":         "usage",
			"finish_reason_path": "choices[0].finish_reason",
		}
	case "audio_tts":
		if platform == service.PlatformKlingAudio {
			return map[string]any{
				"delivery":       "json_url",
				"audio_url_path": "data.task_result.audios[0].url",
				"duration_path":  "data.task_result.audios[0].duration",
				"task_id_path":   "data.task_id",
			}
		}
		return map[string]any{
			"delivery":     "audio_binary",
			"content_type": "audio/wav",
		}
	case "voice_management":
		return map[string]any{
			"delivery":       "json",
			"voice_id_path":  "data.task_result.voices[0].voice_id",
			"trial_url_path": "data.task_result.voices[0].trial_url",
			"task_id_path":   "data.task_id",
		}
	case "voice_catalog":
		return map[string]any{
			"delivery": "json",
		}
	case "video_generation":
		return map[string]any{
			"delivery":         "async_json",
			"task_id_path":     "id",
			"polling_endpoint": "/v1/videos/{id}",
		}
	case "image_generation":
		return publicOpenAIImageResponseContract()
	default:
		return map[string]any{
			"delivery": "json",
		}
	}
}

func publicOpenAIResponsesChatResponseContract() map[string]any {
	return map[string]any{
		"delivery":        "openai_responses_json",
		"stream_delivery": "openai_responses_sse",
		"text_paths":      []string{"output[].content[].text", "output_text"},
		"usage_path":      "usage",
		"status_path":     "status",
		"output_path":     "output",
		"stream_events":   []string{"response.output_text.delta", "response.completed", "response.failed", "response.error"},
	}
}

func publicAnthropicMessagesChatResponseContract() map[string]any {
	return map[string]any{
		"delivery":                "anthropic_messages_json",
		"stream_delivery":         "anthropic_messages_sse",
		"text_paths":              []string{"content[?type=text].text"},
		"usage_path":              "usage",
		"finish_reason_path":      "stop_reason",
		"stream_events":           []string{"message_start", "content_block_start", "content_block_delta", "message_delta", "message_stop"},
		"thinking_block_type":     "thinking",
		"thinking_text_path":      "content[?type=thinking].thinking",
		"thinking_signature_path": "content[?type=thinking].signature",
	}
}

func publicGeminiNativeChatResponseContract() map[string]any {
	return map[string]any{
		"delivery":                  "gemini_generate_content_json",
		"stream_delivery":           "gemini_sse",
		"text_paths":                []string{"candidates[].content.parts[].text"},
		"usage_path":                "usageMetadata",
		"finish_reason_path":        "candidates[].finishReason",
		"stream_events":             []string{"data"},
		"stream_text_path":          "data.candidates[].content.parts[].text",
		"stream_finish_reason_path": "data.candidates[].finishReason",
		"thinking_tokens_path":      "usageMetadata.thoughtsTokenCount",
	}
}

func publicOpenAIImageResponseContract() map[string]any {
	return map[string]any{
		"delivery":        "openai_image_json",
		"image_data_path": "data[].b64_json",
		"image_mime_type": "image/png",
		"usage_path":      "usage",
	}
}

func publicGeminiImageResponseContract() map[string]any {
	return map[string]any{
		"delivery":             "gemini_generate_content_json",
		"image_data_path":      "candidates[].content.parts[].inlineData.data",
		"image_mime_type_path": "candidates[].content.parts[].inlineData.mimeType",
		"text_paths":           []string{"candidates[].content.parts[].text"},
		"usage_path":           "usageMetadata",
		"finish_reason_path":   "candidates[].finishReason",
	}
}

func publicStreamContractForPricedModel(protocol, endpoint, streamEndpoint, modelType, operationType string) map[string]any {
	if streamEndpoint == "" || modelType != "chat" {
		return nil
	}
	switch protocol {
	case "openai":
		if strings.Contains(endpoint, "/v1/responses") {
			return map[string]any{
				"supported":        true,
				"endpoint":         streamEndpoint,
				"request_field":    "stream",
				"request_value":    true,
				"delivery":         "sse",
				"content_type":     "text/event-stream",
				"text_delta_event": "response.output_text.delta",
				"text_delta_path":  "delta",
				"done_events":      []string{"response.completed"},
				"error_events":     []string{"response.failed", "response.error", "error"},
				"usage_path":       "response.usage",
			}
		}
	case "anthropic":
		return map[string]any{
			"supported":           true,
			"endpoint":            streamEndpoint,
			"request_field":       "stream",
			"request_value":       true,
			"delivery":            "sse",
			"content_type":        "text/event-stream",
			"text_delta_event":    "content_block_delta",
			"text_delta_path":     "delta.text",
			"thinking_delta_path": "delta.thinking",
			"done_events":         []string{"message_stop"},
			"error_events":        []string{"error"},
			"usage_path":          "message.usage",
			"finish_reason_path":  "delta.stop_reason",
		}
	case "gemini":
		return map[string]any{
			"supported":          true,
			"endpoint":           streamEndpoint,
			"request_field":      "endpoint",
			"delivery":           "sse",
			"content_type":       "text/event-stream",
			"text_delta_event":   "data",
			"text_delta_path":    "candidates[].content.parts[].text",
			"done_events":        []string{"data with candidates[].finishReason"},
			"error_events":       []string{"error"},
			"usage_path":         "usageMetadata",
			"finish_reason_path": "candidates[].finishReason",
		}
	}
	return nil
}

func publicToolContractForPricedModel(protocol, endpoint, modelType, operationType string) map[string]any {
	if modelType != "chat" {
		return nil
	}
	switch protocol {
	case "openai":
		if strings.Contains(endpoint, "/v1/responses") {
			return map[string]any{
				"supported":        true,
				"types":            []string{"function"},
				"request_path":     "tools[]",
				"tool_choice_path": "tool_choice",
				"response_path":    "output[?type=function_call]",
				"fields": map[string]any{
					"name_path":      "output[?type=function_call].name",
					"arguments_path": "output[?type=function_call].arguments",
					"call_id_path":   "output[?type=function_call].call_id",
				},
				"unsupported_types": []string{"image_generation"},
			}
		}
	case "anthropic":
		return map[string]any{
			"supported":        true,
			"types":            []string{"tool_use"},
			"request_path":     "tools[]",
			"tool_choice_path": "tool_choice",
			"response_path":    "content[?type=tool_use]",
			"fields": map[string]any{
				"id_path":    "content[?type=tool_use].id",
				"name_path":  "content[?type=tool_use].name",
				"input_path": "content[?type=tool_use].input",
			},
		}
	case "gemini":
		return map[string]any{
			"supported":        true,
			"types":            []string{"functionDeclarations"},
			"request_path":     "tools[].functionDeclarations[]",
			"tool_choice_path": "toolConfig.functionCallingConfig",
			"response_path":    "candidates[].content.parts[].functionCall",
			"fields": map[string]any{
				"name_path": "candidates[].content.parts[].functionCall.name",
				"args_path": "candidates[].content.parts[].functionCall.args",
			},
		}
	}
	return nil
}

func publicThinkingContractForPricedModel(protocol, endpoint, modelType, operationType string) map[string]any {
	if modelType != "chat" {
		return nil
	}
	switch protocol {
	case "openai":
		if strings.Contains(endpoint, "/v1/responses") {
			return map[string]any{
				"request_supported": true,
				"request_path":      "reasoning",
				"visible_content":   false,
				"usage_tokens_path": "usage.output_tokens_details.reasoning_tokens",
				"notes":             "reasoning parameter is accepted, but verified responses did not expose visible reasoning content",
			}
		}
	case "anthropic":
		return map[string]any{
			"request_supported": true,
			"request_path":      "thinking",
			"visible_content":   true,
			"content_path":      "content[?type=thinking].thinking",
			"signature_path":    "content[?type=thinking].signature",
			"text_may_be_empty": true,
		}
	case "gemini":
		return map[string]any{
			"request_supported": true,
			"request_path":      "generationConfig.thinkingConfig",
			"visible_content":   true,
			"content_path":      "candidates[].content.parts[?thought=true].text",
			"signature_path":    "candidates[].content.parts[].thoughtSignature",
			"usage_tokens_path": "usageMetadata.thoughtsTokenCount",
		}
	}
	return nil
}

func publicMediaContractForPricedModel(protocol, endpoint, modelType, operationType string) map[string]any {
	switch operationType {
	case "image_generation":
		if protocol == "gemini" {
			return map[string]any{
				"kind":           "image",
				"delivery":       "json_base64",
				"data_path":      "candidates[].content.parts[].inlineData.data",
				"mime_type_path": "candidates[].content.parts[].inlineData.mimeType",
				"size_request_paths": []string{
					"generationConfig.imageConfig.aspectRatio",
					"generationConfig.imageConfig.imageSize",
				},
			}
		}
		return map[string]any{
			"kind":                       "image",
			"delivery":                   "json_base64",
			"data_path":                  "data[].b64_json",
			"mime_type":                  "image/png",
			"size_request_path":          "size",
			"supported_response_formats": []string{"b64_json"},
		}
	case "chat_audio":
		return map[string]any{
			"kind":            "audio",
			"delivery":        "json_base64",
			"data_path":       "choices[0].message.audio.data",
			"transcript_path": "choices[0].message.audio.transcript",
			"verified_format": "wav",
		}
	case "audio_tts":
		return map[string]any{
			"kind":            "audio",
			"delivery":        "binary",
			"content_type":    "audio/wav",
			"verified_format": "wav",
		}
	case "video_generation":
		return map[string]any{
			"kind":     "video",
			"delivery": "async_or_binary",
		}
	}
	return nil
}

func publicPollingContractForPricedModel(endpoint, operationType string) map[string]any {
	if operationType != "video_generation" {
		return nil
	}
	return map[string]any{
		"supported":         true,
		"task_id_path":      "id",
		"status_path":       "status",
		"polling_endpoint":  "/v1/videos/{id}",
		"content_endpoint":  "/v1/videos/{id}/content",
		"terminal_statuses": []string{"completed", "failed", "cancelled"},
	}
}

func publicUnsupportedCapabilitiesForPricedModel(protocol, endpoint, modelType, operationType string) []string {
	switch {
	case protocol == "openai" && strings.Contains(endpoint, "/v1/responses"):
		return []string{"visible_reasoning_content", "image_generation_tool"}
	case protocol == "openai" && strings.Contains(endpoint, "/v1/images/generations"):
		return []string{"response_format:url", "sse_streaming"}
	case operationType == "chat_audio":
		return []string{"sse_streaming"}
	default:
		return nil
	}
}

func modelNamesFromPricedDetails(details []service.GatewayPricedModelDetail) []string {
	models := make([]string, 0, len(details))
	for _, detail := range details {
		if strings.TrimSpace(detail.Name) == "" {
			continue
		}
		models = append(models, detail.Name)
	}
	return models
}

func filterPricedModelDetailsByIDs(details []service.GatewayPricedModelDetail, modelIDs []string) []service.GatewayPricedModelDetail {
	if len(details) == 0 || len(modelIDs) == 0 {
		return nil
	}
	byID := make(map[string]service.GatewayPricedModelDetail, len(details))
	for _, detail := range details {
		byID[strings.ToLower(detail.Name)] = detail
	}
	out := make([]service.GatewayPricedModelDetail, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		if detail, ok := byID[strings.ToLower(modelID)]; ok {
			out = append(out, detail)
		}
	}
	return out
}

func writeModelsList(c *gin.Context, modelIDs []string) {
	models := make([]claude.Model, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		models = append(models, claude.Model{
			ID:          modelID,
			Type:        "model",
			DisplayName: modelID,
			CreatedAt:   "2024-01-01T00:00:00Z",
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

func writeCustomModelsList(c *gin.Context, platform string, modelIDs []string) {
	if platform == service.PlatformOpenAI {
		writeOpenAIModelsList(c, modelIDs)
		return
	}
	writeModelsList(c, modelIDs)
}

func writeOpenAIModelsList(c *gin.Context, modelIDs []string) {
	defaultsByID := make(map[string]openai.Model, len(openai.DefaultModels))
	for _, model := range openai.DefaultModels {
		defaultsByID[model.ID] = model
	}

	models := make([]openai.Model, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		if model, ok := defaultsByID[modelID]; ok {
			models = append(models, model)
			continue
		}
		models = append(models, openai.Model{
			ID:          modelID,
			Object:      "model",
			Created:     1704067200,
			OwnedBy:     "openai",
			Type:        "model",
			DisplayName: modelID,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

func customModelsListSource(platform string, availableModels, fallbackModels []string) []string {
	platform = service.NormalizePlatformSlug(platform)
	if platform == service.PlatformAnthropic && len(availableModels) > 0 {
		return mergeModelIDs(availableModels, fallbackModels)
	}
	return availableModels
}

func customModelsListFallbackModels(platform string) []string {
	switch service.NormalizePlatformSlug(platform) {
	case service.PlatformAnthropic, service.PlatformGemini, service.PlatformAntigravity, service.PlatformGrok:
		return defaultModelIDsForPlatform(platform)
	default:
		return nil
	}
}

func filterModelsByCustomList(availableModels, fallbackModels, selectedModels []string) []string {
	if len(selectedModels) == 0 {
		return availableModels
	}
	source := availableModels
	if len(source) == 0 {
		source = fallbackModels
	}
	if len(source) == 0 {
		return nil
	}

	allowed := make([]string, 0, len(source))
	for _, model := range source {
		model = strings.TrimSpace(model)
		if model != "" {
			allowed = append(allowed, model)
		}
	}

	seen := make(map[string]struct{}, len(selectedModels))
	filtered := make([]string, 0, len(selectedModels))
	for _, model := range selectedModels {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if !customModelsListAllowsModel(allowed, model) {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		filtered = append(filtered, model)
	}
	return filtered
}

func customModelsListAllowsModel(availablePatterns []string, model string) bool {
	for _, pattern := range availablePatterns {
		if pattern == model {
			return true
		}
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(model, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}

func defaultModelIDsForPlatform(platform string) []string {
	switch service.NormalizePlatformSlug(platform) {
	case service.PlatformOpenAI:
		return openai.DefaultModelIDs()
	case service.PlatformGemini:
		ids := make([]string, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			ids = append(ids, model.ID)
		}
		return ids
	case service.PlatformAntigravity:
		models := antigravity.DefaultModels()
		ids := make([]string, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.ID)
		}
		return ids
	case service.PlatformAnthropic:
		ids := make([]string, 0, len(claude.DefaultModels)+len(antigravity.DefaultModels()))
		for _, model := range claude.DefaultModels {
			ids = append(ids, model.ID)
		}
		for _, model := range antigravity.DefaultModels() {
			ids = append(ids, model.ID)
		}
		return mergeModelIDs(ids, nil)
	case service.PlatformGrok:
		return xai.DefaultModelIDs()
	default:
		ids := make([]string, 0, len(claude.DefaultModels))
		for _, model := range claude.DefaultModels {
			ids = append(ids, model.ID)
		}
		return ids
	}
}

func mergeModelIDs(primary, secondary []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	merged := make([]string, 0, len(primary)+len(secondary))
	for _, models := range [][]string{primary, secondary} {
		for _, model := range models {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			if _, ok := seen[model]; ok {
				continue
			}
			seen[model] = struct{}{}
			merged = append(merged, model)
		}
	}
	return merged
}

// AntigravityModels 返回 Antigravity 支持的全部模型
// GET /antigravity/models
func (h *GatewayHandler) AntigravityModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   antigravity.DefaultModels(),
	})
}

func cloneAPIKeyWithGroup(apiKey *service.APIKey, group *service.Group) *service.APIKey {
	if apiKey == nil || group == nil {
		return apiKey
	}
	cloned := *apiKey
	groupID := group.ID
	cloned.GroupID = &groupID
	cloned.Group = group
	return &cloned
}

func (h *GatewayHandler) isGeminiProtocolPlatform(ctx context.Context, platform string) bool {
	platform = service.NormalizePlatformSlug(platform)
	if platform == service.PlatformGemini {
		return true
	}
	if platform == "" || h == nil || h.platformService == nil {
		return false
	}
	return h.platformService.IsGeminiCompatible(ctx, platform)
}

// Usage handles getting account balance and usage statistics for CC Switch integration
// GET /v1/usage
//
// Two modes:
//   - quota_limited: API Key has quota or rate limits configured. Returns key-level limits/usage.
//   - unrestricted:  No key-level limits. Returns subscription or wallet balance info.
func (h *GatewayHandler) Usage(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	ctx := c.Request.Context()

	// 解析可选的日期范围参数（用于 model_stats 查询）
	startTime, endTime := h.parseUsageDateRange(c)
	days, ok := parseAPIKeyDailyUsageDays(c.DefaultQuery("days", ""))
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Invalid days, allowed range is 1-90")
		return
	}

	// Best-effort: 获取用量统计（按当前 API Key 过滤），失败不影响基础响应
	usageData := h.buildUsageData(ctx, apiKey.ID)
	dailyUsage := h.buildAPIKeyDailyUsage(c, subject.UserID, apiKey.ID, days)

	// Best-effort: 获取模型统计
	var modelStats any
	if h.usageService != nil {
		if stats, err := h.usageService.GetAPIKeyModelStats(ctx, apiKey.ID, startTime, endTime); err == nil && len(stats) > 0 {
			modelStats = stats
		}
	}

	// 判断模式: key 有总额度或速率限制 → quota_limited，否则 → unrestricted
	isQuotaLimited := apiKey.Quota > 0 || apiKey.HasRateLimits()

	if isQuotaLimited {
		h.usageQuotaLimited(c, ctx, apiKey, usageData, dailyUsage, modelStats)
		return
	}

	h.usageUnrestricted(c, ctx, apiKey, subject, usageData, dailyUsage, modelStats)
}

// parseUsageDateRange 解析 start_date / end_date query params，默认返回近 30 天范围
func (h *GatewayHandler) parseUsageDateRange(c *gin.Context) (time.Time, time.Time) {
	now := timezone.Now()
	endTime := now
	startTime := now.AddDate(0, 0, -30)

	if s := c.Query("start_date"); s != "" {
		if t, err := timezone.ParseInLocation("2006-01-02", s); err == nil {
			startTime = t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := timezone.ParseInLocation("2006-01-02", s); err == nil {
			endTime = t.AddDate(0, 0, 1) // half-open range upper bound
		}
	}
	return startTime, endTime
}

// buildUsageData 构建 today/total 用量摘要
func (h *GatewayHandler) buildUsageData(ctx context.Context, apiKeyID int64) gin.H {
	if h.usageService == nil {
		return nil
	}
	dashStats, err := h.usageService.GetAPIKeyDashboardStats(ctx, apiKeyID)
	if err != nil || dashStats == nil {
		return nil
	}
	return gin.H{
		"today": gin.H{
			"requests":              dashStats.TodayRequests,
			"input_tokens":          dashStats.TodayInputTokens,
			"output_tokens":         dashStats.TodayOutputTokens,
			"cache_creation_tokens": dashStats.TodayCacheCreationTokens,
			"cache_read_tokens":     dashStats.TodayCacheReadTokens,
			"total_tokens":          dashStats.TodayTokens,
			"cost":                  dashStats.TodayCost,
			"actual_cost":           dashStats.TodayActualCost,
		},
		"total": gin.H{
			"requests":              dashStats.TotalRequests,
			"input_tokens":          dashStats.TotalInputTokens,
			"output_tokens":         dashStats.TotalOutputTokens,
			"cache_creation_tokens": dashStats.TotalCacheCreationTokens,
			"cache_read_tokens":     dashStats.TotalCacheReadTokens,
			"total_tokens":          dashStats.TotalTokens,
			"cost":                  dashStats.TotalCost,
			"actual_cost":           dashStats.TotalActualCost,
		},
		"average_duration_ms": dashStats.AverageDurationMs,
		"rpm":                 dashStats.Rpm,
		"tpm":                 dashStats.Tpm,
	}
}

func (h *GatewayHandler) buildAPIKeyDailyUsage(c *gin.Context, userID, apiKeyID int64, days int) any {
	if h.usageService == nil {
		return nil
	}
	startTime, endTime := apiKeyDailyUsageRange(days, c.Query("timezone"))
	stats, err := h.usageService.GetAPIKeyDailyUsage(c.Request.Context(), userID, apiKeyID, startTime, endTime)
	if err != nil {
		return nil
	}
	return stats
}

// usageQuotaLimited 处理 quota_limited 模式的响应
func (h *GatewayHandler) usageQuotaLimited(c *gin.Context, ctx context.Context, apiKey *service.APIKey, usageData gin.H, dailyUsage any, modelStats any) {
	resp := gin.H{
		"mode":    "quota_limited",
		"isValid": apiKey.Status == service.StatusAPIKeyActive || apiKey.Status == service.StatusAPIKeyQuotaExhausted || apiKey.Status == service.StatusAPIKeyExpired,
		"status":  apiKey.Status,
	}

	// 总额度信息
	if apiKey.Quota > 0 {
		remaining := apiKey.GetQuotaRemaining()
		resp["quota"] = gin.H{
			"limit":     apiKey.Quota,
			"used":      apiKey.QuotaUsed,
			"remaining": remaining,
			"unit":      "USD",
		}
		resp["remaining"] = remaining
		resp["unit"] = "USD"
	}

	// 速率限制信息（从 DB 获取实时用量）
	if apiKey.HasRateLimits() && h.apiKeyService != nil {
		rateLimitData, err := h.apiKeyService.GetRateLimitData(ctx, apiKey.ID)
		if err == nil && rateLimitData != nil {
			var rateLimits []gin.H
			if apiKey.RateLimit5h > 0 {
				used := rateLimitData.EffectiveUsage5h()
				entry := gin.H{
					"window":       "5h",
					"limit":        apiKey.RateLimit5h,
					"used":         used,
					"remaining":    max(0, apiKey.RateLimit5h-used),
					"window_start": rateLimitData.Window5hStart,
				}
				if rateLimitData.Window5hStart != nil && !service.IsWindowExpired(rateLimitData.Window5hStart, service.RateLimitWindow5h) {
					entry["reset_at"] = rateLimitData.Window5hStart.Add(service.RateLimitWindow5h)
				}
				rateLimits = append(rateLimits, entry)
			}
			if apiKey.RateLimit1d > 0 {
				used := rateLimitData.EffectiveUsage1d()
				entry := gin.H{
					"window":       "1d",
					"limit":        apiKey.RateLimit1d,
					"used":         used,
					"remaining":    max(0, apiKey.RateLimit1d-used),
					"window_start": rateLimitData.Window1dStart,
				}
				if rateLimitData.Window1dStart != nil && !service.IsWindowExpired(rateLimitData.Window1dStart, service.RateLimitWindow1d) {
					entry["reset_at"] = rateLimitData.Window1dStart.Add(service.RateLimitWindow1d)
				}
				rateLimits = append(rateLimits, entry)
			}
			if apiKey.RateLimit7d > 0 {
				used := rateLimitData.EffectiveUsage7d()
				entry := gin.H{
					"window":       "7d",
					"limit":        apiKey.RateLimit7d,
					"used":         used,
					"remaining":    max(0, apiKey.RateLimit7d-used),
					"window_start": rateLimitData.Window7dStart,
				}
				if rateLimitData.Window7dStart != nil && !service.IsWindowExpired(rateLimitData.Window7dStart, service.RateLimitWindow7d) {
					entry["reset_at"] = rateLimitData.Window7dStart.Add(service.RateLimitWindow7d)
				}
				rateLimits = append(rateLimits, entry)
			}
			if len(rateLimits) > 0 {
				resp["rate_limits"] = rateLimits
			}
		}
	}

	// 过期时间
	if apiKey.ExpiresAt != nil {
		resp["expires_at"] = apiKey.ExpiresAt
		resp["days_until_expiry"] = apiKey.GetDaysUntilExpiry()
	}

	if usageData != nil {
		resp["usage"] = usageData
	}
	if dailyUsage != nil {
		resp["daily_usage"] = dailyUsage
	}
	if modelStats != nil {
		resp["model_stats"] = modelStats
	}

	c.JSON(http.StatusOK, resp)
}

// usageUnrestricted 处理 unrestricted 模式的响应（向后兼容）
func (h *GatewayHandler) usageUnrestricted(c *gin.Context, ctx context.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, usageData gin.H, dailyUsage any, modelStats any) {
	// 订阅模式
	if apiKey.Group != nil && apiKey.Group.IsSubscriptionType() {
		resp := gin.H{
			"mode":     "unrestricted",
			"isValid":  true,
			"planName": apiKey.Group.Name,
			"unit":     "USD",
		}

		// 订阅信息可能不在 context 中（/v1/usage 路径跳过了中间件的计费检查）
		subscription, ok := middleware2.GetSubscriptionFromContext(c)
		if ok {
			remaining := h.calculateSubscriptionRemaining(apiKey.Group, subscription)
			resp["remaining"] = remaining
			resp["subscription"] = gin.H{
				"daily_usage_usd":     subscription.DailyUsageUSD,
				"weekly_usage_usd":    subscription.WeeklyUsageUSD,
				"monthly_usage_usd":   subscription.MonthlyUsageUSD,
				"daily_limit_usd":     apiKey.Group.DailyLimitUSD,
				"weekly_limit_usd":    apiKey.Group.WeeklyLimitUSD,
				"monthly_limit_usd":   apiKey.Group.MonthlyLimitUSD,
				"weekly_window_start": subscription.WeeklyWindowStart,
				"expires_at":          subscription.ExpiresAt,
			}
		}

		if usageData != nil {
			resp["usage"] = usageData
		}
		if dailyUsage != nil {
			resp["daily_usage"] = dailyUsage
		}
		if modelStats != nil {
			resp["model_stats"] = modelStats
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	// 余额模式
	latestUser, err := h.userService.GetByID(ctx, subject.UserID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to get user info")
		return
	}

	resp := gin.H{
		"mode":      "unrestricted",
		"isValid":   true,
		"planName":  "钱包余额",
		"remaining": latestUser.Balance,
		"unit":      "USD",
		"balance":   latestUser.Balance,
	}
	if usageData != nil {
		resp["usage"] = usageData
	}
	if dailyUsage != nil {
		resp["daily_usage"] = dailyUsage
	}
	if modelStats != nil {
		resp["model_stats"] = modelStats
	}
	c.JSON(http.StatusOK, resp)
}

// calculateSubscriptionRemaining 计算订阅剩余可用额度
// 逻辑：
// 1. 如果日/周/月任一限额达到100%，返回0
// 2. 否则返回所有已配置周期中剩余额度的最小值
func (h *GatewayHandler) calculateSubscriptionRemaining(group *service.Group, sub *service.UserSubscription) float64 {
	var remainingValues []float64

	// 检查日限额
	if group.HasDailyLimit() {
		remaining := *group.DailyLimitUSD - sub.DailyUsageUSD
		if remaining <= 0 {
			return 0
		}
		remainingValues = append(remainingValues, remaining)
	}

	// 检查周限额
	if group.HasWeeklyLimit() {
		remaining := *group.WeeklyLimitUSD - sub.WeeklyUsageUSD
		if remaining <= 0 {
			return 0
		}
		remainingValues = append(remainingValues, remaining)
	}

	// 检查月限额
	if group.HasMonthlyLimit() {
		remaining := *group.MonthlyLimitUSD - sub.MonthlyUsageUSD
		if remaining <= 0 {
			return 0
		}
		remainingValues = append(remainingValues, remaining)
	}

	// 如果没有配置任何限额，返回-1表示无限制
	if len(remainingValues) == 0 {
		return -1
	}

	// 返回最小值
	min := remainingValues[0]
	for _, v := range remainingValues[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// handleConcurrencyError handles concurrency-related acquire errors.
func (h *GatewayHandler) handleConcurrencyError(c *gin.Context, err error, slotType string, streamStarted bool) {
	status, errType, message := concurrencyErrorResponse(err, slotType)
	h.handleStreamingAwareError(c, status, errType, message, streamStarted)
}

func (h *GatewayHandler) handleFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError, platform string, streamStarted bool) {
	statusCode := failoverErr.StatusCode
	responseBody := failoverErr.ResponseBody
	if service.IsOpenAISilentRefusalErrorBody(responseBody) {
		service.SetOpsUpstreamError(c, statusCode, service.OpenAISilentRefusalClientMessage(), "")
		h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", service.OpenAISilentRefusalClientMessage(), streamStarted)
		return
	}

	// 先检查透传规则
	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule(platform, statusCode, responseBody); rule != nil {
			// 确定响应状态码
			respCode := statusCode
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				respCode = *rule.ResponseCode
			}

			// 确定响应消息
			msg := service.ExtractUpstreamErrorMessage(responseBody)
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				msg = *rule.CustomMessage
			}

			if rule.SkipMonitoring {
				c.Set(service.OpsSkipPassthroughKey, true)
			}

			h.handleStreamingAwareError(c, respCode, "upstream_error", msg, streamStarted)
			return
		}
	}

	// 记录原始上游状态码，以便 ops 错误日志捕获真实的上游错误
	upstreamMsg := service.ExtractUpstreamErrorMessage(responseBody)
	service.SetOpsUpstreamError(c, statusCode, upstreamMsg, "")

	// 使用默认的错误映射
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	h.handleStreamingAwareError(c, status, errType, errMsg, streamStarted)
}

// handleFailoverExhaustedSimple 简化版本，用于没有响应体的情况
func (h *GatewayHandler) handleFailoverExhaustedSimple(c *gin.Context, statusCode int, streamStarted bool) {
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	service.SetOpsUpstreamError(c, statusCode, errMsg, "")
	h.handleStreamingAwareError(c, status, errType, errMsg, streamStarted)
}

func (h *GatewayHandler) mapUpstreamError(statusCode int) (int, string, string) {
	switch statusCode {
	case 401:
		return http.StatusBadGateway, "upstream_error", "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "upstream_error", "Upstream access forbidden, please contact administrator"
	case 429:
		return http.StatusTooManyRequests, "rate_limit_error", "Upstream rate limit exceeded, please retry later"
	case 529:
		return http.StatusServiceUnavailable, "overloaded_error", "Upstream service overloaded, please retry later"
	case 500, 502, 503, 504:
		return http.StatusBadGateway, "upstream_error", "Upstream service temporarily unavailable"
	default:
		return http.StatusBadGateway, "upstream_error", "Upstream request failed"
	}
}

// handleStreamingAwareError handles errors that may occur after streaming has started
func (h *GatewayHandler) handleStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	if streamStarted {
		// 响应状态码已固化为 200（ping/部分数据已 flush），错误只能就地以 SSE 帧回传。
		// 标记本次流内错误，供 ops_error_logger 补记——否则该中间件按 status>=400 采集，
		// 这类挂在 200 流上的失败（如并发限流回退）不会进错误看板。
		service.MarkOpsStreamError(c, errType, message, status)

		// /v1/responses 的严格 SDK（Codex CLI）要求终止事件必须属于
		// response.completed/failed/incomplete/cancelled 集合。
		// Anthropic-backed Responses 路径同样会因为通用 error 帧被拒。
		if inboundIsResponses(c) {
			if writeResponsesFailedSSE(c, errType, message) {
				return
			}
		}
		// Stream already started, send error as SSE event then close
		flusher, ok := c.Writer.(http.Flusher)
		if ok {
			// SSE 错误事件固定 schema，使用 Quote 直拼可避免额外 Marshal 分配。
			errorEvent := `data: {"type":"error","error":{"type":` + strconv.Quote(errType) + `,"message":` + strconv.Quote(message) + `}}` + "\n\n"
			if _, err := fmt.Fprint(c.Writer, errorEvent); err != nil {
				_ = c.Error(err)
			}
			flusher.Flush()
		}
		return
	}

	// Normal case: return JSON response with proper status code
	h.errorResponse(c, status, errType, message)
}

// ensureForwardErrorResponse 在 Forward 返回错误但尚未写响应时补写统一错误响应。
// Writer 已被写过时（ping 已 flush）走 streamStarted 分支，
// 让 handleStreamingAwareError 通过 SSE 发协议合规的终止事件，
// 否则下游收到的就是 silent EOF。
func (h *GatewayHandler) ensureForwardErrorResponse(c *gin.Context, streamStarted bool) bool {
	if c == nil || c.Writer == nil {
		return false
	}
	if service.IsResponseCommitted(c) {
		return false
	}
	if c.Writer.Written() {
		streamStarted = true
	}
	h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", streamStarted)
	return true
}

// gatewayForwardErrorAlreadyCommunicated reports whether a Forward implementation
// has already written a complete error response to the client before returning
// an error to the handler.
//
// This is intentionally narrower than "writer size changed": a stream may have
// only emitted keepalive pings or partial data, in which case the handler still
// needs to append a protocol-level terminal error. Non-SSE output from Forward
// is different: service-level helpers such as handleErrorResponse/writeClaudeError
// already wrote the client-visible JSON body, so adding the generic streaming
// fallback would corrupt the response by appending a second `data: ...` frame.
func gatewayForwardErrorAlreadyCommunicated(c *gin.Context, writerSizeBeforeForward int, err error) bool {
	if err == nil || c == nil || c.Writer == nil {
		return false
	}
	if c.Writer.Size() == writerSizeBeforeForward {
		return false
	}

	contentType := strings.ToLower(strings.TrimSpace(c.Writer.Header().Get("Content-Type")))
	if contentType == "" {
		return false
	}
	return !strings.Contains(contentType, "text/event-stream")
}

// checkClaudeCodeVersion 检查 Claude Code 客户端版本是否满足版本要求
// 仅对已识别的 Claude Code 客户端执行，count_tokens 路径除外
func (h *GatewayHandler) checkClaudeCodeVersion(c *gin.Context) bool {
	ctx := c.Request.Context()
	if !service.IsClaudeCodeClient(ctx) {
		return true
	}

	// 排除 count_tokens 子路径
	if strings.HasSuffix(c.Request.URL.Path, "/count_tokens") {
		return true
	}

	minVersion, maxVersion := h.settingService.GetClaudeCodeVersionBounds(ctx)
	if minVersion == "" && maxVersion == "" {
		return true // 未设置，不检查
	}

	clientVersion := service.GetClaudeCodeVersion(ctx)
	if clientVersion == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error",
			"Unable to determine Claude Code version. Please update Claude Code: npm update -g @anthropic-ai/claude-code")
		return false
	}

	if minVersion != "" && service.CompareVersions(clientVersion, minVersion) < 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Your Claude Code version (%s) is below the minimum required version (%s). Please update: npm update -g @anthropic-ai/claude-code",
				clientVersion, minVersion))
		return false
	}

	if maxVersion != "" && service.CompareVersions(clientVersion, maxVersion) > 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Your Claude Code version (%s) exceeds the maximum allowed version (%s). "+
				"Please downgrade: npm install -g @anthropic-ai/claude-code@%s && "+
				"set CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1 to prevent auto-upgrade",
				clientVersion, maxVersion, maxVersion))
		return false
	}

	return true
}

// errorResponse 返回Claude API格式的错误响应
func (h *GatewayHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

// CountTokens handles token counting endpoint
// POST /v1/messages/count_tokens
// 特点：校验订阅/余额，但不计算并发、不记录使用量
func (h *GatewayHandler) CountTokens(c *gin.Context) {
	// 从context获取apiKey和user（ApiKeyAuth中间件已设置）
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	_, ok = middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.gateway.count_tokens",
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	defer h.maybeLogCompatibilityFallbackMetrics(reqLog)

	// 读取请求体
	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
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

	setOpsRequestContext(c, "", false)

	bodyRef := service.NewRequestBodyRef(body)
	parsedReq, err := service.ParseGatewayRequest(bodyRef, domain.PlatformAnthropic)
	if err != nil {
		logRequestBodyParseFailure(reqLog, body, err)
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	// count_tokens 走 messages 严格校验时，复用已解析请求，避免二次反序列化。
	SetClaudeCodeClientContext(c, body, parsedReq)
	reqLog = reqLog.With(zap.String("model", parsedReq.Model), zap.Bool("stream", parsedReq.Stream))
	// 在请求上下文中记录 thinking 状态，供 Antigravity 最终模型 key 推导/模型维度限流使用
	c.Request = c.Request.WithContext(service.WithThinkingEnabled(c.Request.Context(), parsedReq.ThinkingEnabled, h.metadataBridgeEnabled()))

	// 验证 model 必填
	if parsedReq.Model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	setOpsRequestContext(c, parsedReq.Model, parsedReq.Stream)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(parsedReq.Stream, false)))

	// 获取订阅信息（可能为nil）
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	// 校验 billing eligibility（订阅/余额）
	// 【注意】不计算并发，但需要校验订阅/余额
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.errorResponse(c, status, code, message)
		return
	}

	// 计算粘性会话 hash
	parsedReq.SessionContext = &service.SessionContext{
		ClientIP:  ip.GetClientIP(c),
		UserAgent: c.GetHeader("User-Agent"),
		APIKeyID:  apiKey.ID,
	}
	sessionHash := h.gatewayService.GenerateSessionHash(parsedReq)

	// 选择支持该模型的账号
	account, err := h.gatewayService.SelectAccountForModel(c.Request.Context(), apiKey.GroupID, sessionHash, parsedReq.Model)
	if err != nil {
		reqLog.Warn("gateway.count_tokens_select_account_failed", zap.Error(err))
		cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, parsedReq.Model, parsedReq.Model, service.PlatformAnthropic)
		if !cls.ModelNotFound {
			markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		}
		h.errorResponse(c, cls.Status, cls.ErrType, cls.Message)
		return
	}
	setOpsSelectedAccount(c, account.ID, account.Platform)

	// 转发请求（不记录使用量）
	if err := h.gatewayService.ForwardCountTokens(c.Request.Context(), c, account, parsedReq); err != nil {
		reqLog.Error("gateway.count_tokens_forward_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		// 错误响应已在 ForwardCountTokens 中处理
		return
	}
}

// InterceptType 表示请求拦截类型
type InterceptType int

const (
	InterceptTypeNone              InterceptType = iota
	InterceptTypeWarmup                          // 预热请求（返回 "New Conversation"）
	InterceptTypeSuggestionMode                  // SUGGESTION MODE（返回空字符串）
	InterceptTypeMaxTokensOneHaiku               // max_tokens=1 + haiku 探测请求（返回 "#"）
)

// isHaikuModel 检查模型名称是否包含 "haiku"（大小写不敏感）
func isHaikuModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "haiku")
}

// isMaxTokensOneHaikuRequest 检查是否为 max_tokens=1 + haiku 模型的探测请求
// 这类请求用于 Claude Code 验证 API 连通性（流式/非流式均会出现，如 cc-switch v3.9.0 起的健康检查探测为流式）
// 条件：max_tokens == 1 且 model 包含 "haiku"
func isMaxTokensOneHaikuRequest(model string, maxTokens int) bool {
	return maxTokens == 1 && isHaikuModel(model)
}

// detectInterceptType 检测请求是否需要拦截，返回拦截类型
// 参数说明：
//   - body: 请求体字节
//   - model: 请求的模型名称
//   - maxTokens: max_tokens 值
//   - isClaudeCodeClient: 是否已通过 Claude Code 客户端校验
func detectInterceptType(body []byte, model string, maxTokens int, isClaudeCodeClient bool) InterceptType {
	// 优先检查 max_tokens=1 + haiku 探测请求（流式/非流式均适用）
	if isClaudeCodeClient && isMaxTokensOneHaikuRequest(model, maxTokens) {
		return InterceptTypeMaxTokensOneHaiku
	}

	// 快速检查：如果不包含任何关键字，直接返回
	bodyStr := string(body)
	hasSuggestionMode := strings.Contains(bodyStr, "[SUGGESTION MODE:")
	hasWarmupKeyword := strings.Contains(bodyStr, "title") || strings.Contains(bodyStr, "Warmup")

	if !hasSuggestionMode && !hasWarmupKeyword {
		return InterceptTypeNone
	}

	// 解析请求（只解析一次）
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
		System []struct {
			Text string `json:"text"`
		} `json:"system"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return InterceptTypeNone
	}

	// 检查 SUGGESTION MODE（最后一条 user 消息）
	if hasSuggestionMode && len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" && len(lastMsg.Content) > 0 &&
			lastMsg.Content[0].Type == "text" &&
			strings.HasPrefix(lastMsg.Content[0].Text, "[SUGGESTION MODE:") {
			return InterceptTypeSuggestionMode
		}
	}

	// 检查 Warmup 请求
	if hasWarmupKeyword {
		// 检查 messages 中的标题提示模式
		for _, msg := range req.Messages {
			for _, content := range msg.Content {
				if content.Type == "text" {
					if strings.Contains(content.Text, "Please write a 5-10 word title for the following conversation:") ||
						content.Text == "Warmup" {
						return InterceptTypeWarmup
					}
				}
			}
		}
		// 检查 system 中的标题提取模式
		for _, sys := range req.System {
			if strings.Contains(sys.Text, "nalyze if this message indicates a new conversation topic. If it does, extract a 2-3 word title") {
				return InterceptTypeWarmup
			}
		}
	}

	return InterceptTypeNone
}

// sendMockInterceptStream 发送流式 mock 响应（用于请求拦截）
func sendMockInterceptStream(c *gin.Context, model string, interceptType InterceptType) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 根据拦截类型决定响应内容
	var msgID string
	var outputTokens int
	var textDeltas []string

	switch interceptType {
	case InterceptTypeSuggestionMode:
		msgID = "msg_mock_suggestion"
		outputTokens = 1
		textDeltas = []string{""} // 空内容
	default: // InterceptTypeWarmup
		msgID = "msg_mock_warmup"
		outputTokens = 2
		textDeltas = []string{"New", " Conversation"}
	}

	// Build message_start event with fixed schema.
	messageStartJSON := `{"type":"message_start","message":{"id":` + strconv.Quote(msgID) + `,"type":"message","role":"assistant","model":` + strconv.Quote(model) + `,"content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`

	// Build events
	events := []string{
		`event: message_start` + "\n" + `data: ` + string(messageStartJSON),
		`event: content_block_start` + "\n" + `data: {"content_block":{"text":"","type":"text"},"index":0,"type":"content_block_start"}`,
	}

	// Add text deltas
	for _, text := range textDeltas {
		deltaJSON := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":` + strconv.Quote(text) + `}}`
		events = append(events, `event: content_block_delta`+"\n"+`data: `+string(deltaJSON))
	}

	// Add final events
	messageDeltaJSON := `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_tokens":10,"output_tokens":` + strconv.Itoa(outputTokens) + `}}`

	events = append(events,
		`event: content_block_stop`+"\n"+`data: {"index":0,"type":"content_block_stop"}`,
		`event: message_delta`+"\n"+`data: `+string(messageDeltaJSON),
		`event: message_stop`+"\n"+`data: {"type":"message_stop"}`,
	)

	for _, event := range events {
		_, _ = c.Writer.WriteString(event + "\n\n")
		c.Writer.Flush()
		time.Sleep(20 * time.Millisecond)
	}
}

// generateRealisticMsgID 生成仿真的消息 ID（msg_bdrk_XXXXXXX 格式）
// 格式与 Claude API 真实响应一致，24 位随机字母数字
func generateRealisticMsgID() string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const idLen = 24
	randomBytes := make([]byte, idLen)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Sprintf("msg_bdrk_%d", time.Now().UnixNano())
	}
	b := make([]byte, idLen)
	for i := range b {
		b[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return "msg_bdrk_" + string(b)
}

// sendMockInterceptResponse 发送非流式 mock 响应（用于请求拦截）
func sendMockInterceptResponse(c *gin.Context, model string, interceptType InterceptType) {
	var msgID, text, stopReason string
	var outputTokens int

	switch interceptType {
	case InterceptTypeSuggestionMode:
		msgID = "msg_mock_suggestion"
		text = ""
		outputTokens = 1
		stopReason = "end_turn"
	case InterceptTypeMaxTokensOneHaiku:
		msgID = generateRealisticMsgID()
		text = "#"
		outputTokens = 1
		stopReason = "max_tokens" // max_tokens=1 探测请求的 stop_reason 应为 max_tokens
	default: // InterceptTypeWarmup
		msgID = "msg_mock_warmup"
		text = "New Conversation"
		outputTokens = 2
		stopReason = "end_turn"
	}

	// 构建完整的响应格式（与 Claude API 响应格式一致）
	response := gin.H{
		"model":         model,
		"id":            msgID,
		"type":          "message",
		"role":          "assistant",
		"content":       []gin.H{{"type": "text", "text": text}},
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": gin.H{
			"input_tokens":                10,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
			"cache_creation": gin.H{
				"ephemeral_5m_input_tokens": 0,
				"ephemeral_1h_input_tokens": 0,
			},
			"output_tokens": outputTokens,
			"total_tokens":  10 + outputTokens,
		},
	}

	c.JSON(http.StatusOK, response)
}

// extractQuotaResetSeconds 从 quota 错误的 metadata 中提取 window_resets_at 并计算
// 距重置剩余秒数。fallback 路径必须返回 ≥1 秒，避免客户端立即重试无限循环。
func extractQuotaResetSeconds(err error) int {
	const fallback = 60
	appErr := pkgerrors.FromError(err)
	if appErr == nil {
		return fallback
	}
	raw, ok := appErr.Metadata["window_resets_at"]
	if !ok || raw == "" {
		return fallback
	}
	resetAt, parseErr := time.Parse(time.RFC3339, raw)
	if parseErr != nil {
		logger.L().With(
			zap.String("component", "handler.gateway.billing"),
			zap.String("raw", raw),
			zap.Error(parseErr),
		).Warn("quota.invalid_window_resets_at_format")
		return fallback
	}
	secs := time.Until(resetAt).Seconds()
	if secs <= 0 {
		// reset 时间已过：cache 与 DB 应该正在自愈，返回 fallback 让客户端按常规节奏退避，
		// 避免返回 1 秒导致客户端立即重试仍触发限额的退避循环。
		return fallback
	}
	return int(math.Ceil(secs))
}

func billingErrorDetails(err error) (status int, code, message string, retryAfter int) {
	if errors.Is(err, service.ErrBillingServiceUnavailable) {
		msg := pkgerrors.Message(err)
		if msg == "" {
			msg = "Billing service temporarily unavailable. Please retry later."
		}
		return http.StatusServiceUnavailable, "billing_service_error", msg, 0
	}
	if errors.Is(err, service.ErrAPIKeyRateLimit5hExceeded) {
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg, 0
	}
	if errors.Is(err, service.ErrAPIKeyRateLimit1dExceeded) {
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg, 0
	}
	if errors.Is(err, service.ErrAPIKeyRateLimit7dExceeded) {
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg, 0
	}
	// 用户/分组 RPM 超限统一映射为 HTTP 429；保留与其它 rate_limit 一致的错误码便于客户端分类。
	// 返回 Retry-After 秒数（当前分钟剩余秒数），让 SDK 自动退避。
	if errors.Is(err, service.ErrGroupRPMExceeded) || errors.Is(err, service.ErrUserRPMExceeded) {
		msg := pkgerrors.Message(err)
		retrySeconds := 60 - int(time.Now().Unix()%60)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg, retrySeconds
	}
	if errors.Is(err, service.ErrUserPlatformDailyQuotaExhausted) ||
		errors.Is(err, service.ErrUserPlatformWeeklyQuotaExhausted) ||
		errors.Is(err, service.ErrUserPlatformMonthlyQuotaExhausted) {
		// 与 RPM 超限一致映射 429 + Retry-After，让 SDK 自动退避（而非 403 直接失败）。
		// 错误码用 rate_limit_exceeded 与 OpenAI 兼容客户端一致；细分类型由 ErrCode + window_resets_at metadata 区分。
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg, extractQuotaResetSeconds(err)
	}
	msg := pkgerrors.Message(err)
	if msg == "" {
		logger.L().With(
			zap.String("component", "handler.gateway.billing"),
			zap.Error(err),
		).Warn("gateway.billing_error_missing_message")
		msg = "Billing error"
	}
	return http.StatusForbidden, "billing_error", msg, 0
}

func (h *GatewayHandler) metadataBridgeEnabled() bool {
	if h == nil || h.cfg == nil {
		return true
	}
	return h.cfg.Gateway.OpenAIWS.MetadataBridgeEnabled
}

func (h *GatewayHandler) maybeLogCompatibilityFallbackMetrics(reqLog *zap.Logger) {
	if reqLog == nil {
		return
	}
	if gatewayCompatibilityMetricsLogCounter.Add(1)%gatewayCompatibilityMetricsLogInterval != 0 {
		return
	}
	metrics := service.SnapshotOpenAICompatibilityFallbackMetrics()
	reqLog.Info("gateway.compatibility_fallback_metrics",
		zap.Int64("session_hash_legacy_read_fallback_total", metrics.SessionHashLegacyReadFallbackTotal),
		zap.Int64("session_hash_legacy_read_fallback_hit", metrics.SessionHashLegacyReadFallbackHit),
		zap.Int64("session_hash_legacy_dual_write_total", metrics.SessionHashLegacyDualWriteTotal),
		zap.Float64("session_hash_legacy_read_hit_rate", metrics.SessionHashLegacyReadHitRate),
		zap.Int64("metadata_legacy_fallback_total", metrics.MetadataLegacyFallbackTotal),
	)
}

func (h *GatewayHandler) submitUsageRecordTask(parent context.Context, task service.UsageRecordTask) {
	if task == nil {
		return
	}
	task = wrapUsageRecordTaskContext(parent, task)
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)
		return
	}
	// 回退路径：worker 池未注入时同步执行，避免退回到无界 goroutine 模式。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().With(
				zap.String("component", "handler.gateway.messages"),
				zap.Any("panic", recovered),
			).Error("gateway.usage_record_task_panic_recovered")
		}
	}()
	task(ctx)
}

// getUserMsgQueueMode 获取当前请求的 UMQ 模式
// 返回 "serialize" | "throttle" | ""
func (h *GatewayHandler) getUserMsgQueueMode(account *service.Account, parsed *service.ParsedRequest) string {
	if h.userMsgQueueHelper == nil {
		return ""
	}
	// 仅适用于 Anthropic OAuth/SetupToken 账号
	if !account.IsAnthropicOAuthOrSetupToken() {
		return ""
	}
	if !service.IsRealUserMessage(parsed) {
		return ""
	}
	// 账号级模式优先，fallback 到全局配置
	mode := account.GetUserMsgQueueMode()
	if mode == "" {
		mode = h.cfg.Gateway.UserMessageQueue.GetEffectiveMode()
	}
	return mode
}
