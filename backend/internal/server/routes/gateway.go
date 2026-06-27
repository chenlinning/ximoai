package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 注册 API 网关路由（Claude/OpenAI/Gemini 兼容）
func RegisterGatewayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	platformService *service.PlatformService,
	cfg *config.Config,
) {
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()

	// 未分组 Key 拦截中间件（按协议格式区分错误响应）
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)

	rejectGrokUnsupportedEndpoint := func(c *gin.Context, endpoint string) {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": endpoint + " is not supported for Grok groups",
			},
		})
	}

	// API网关（Claude API兼容）
	gateway := r.Group("/v1")
	gateway.Use(bodyLimit)
	gateway.Use(clientRequestID)
	gateway.Use(opsErrorLogger)
	gateway.Use(endpointNorm)
	gateway.Use(gin.HandlerFunc(apiKeyAuth))
	gateway.Use(requireGroupAnthropic)
	{
		// /v1/messages: auto-route based on group platform
		gateway.POST("/messages", func(c *gin.Context) {
			if rejectOpenAICompatibleMissingCapability(c, platformService, service.PlatformCapabilityResponses, "Responses API is not supported for this platform") {
				return
			}
			if getGroupPlatform(c) == service.PlatformGrok {
				rejectGrokUnsupportedEndpoint(c, "Messages API")
				return
			}
			if isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityResponses) {
				h.OpenAIGateway.Messages(c)
				return
			}
			h.Gateway.Messages(c)
		})
		// /v1/messages/count_tokens: OpenAI groups get 404
		gateway.POST("/messages/count_tokens", func(c *gin.Context) {
			if isGroupOpenAICompatible(c, platformService) || isGroupGeminiCompatible(c, platformService) {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"type": "error",
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Token counting is not supported for this platform",
					},
				})
				return
			}
			h.Gateway.CountTokens(c)
		})
		gateway.GET("/models", h.Gateway.Models)
		gateway.GET("/usage", h.Gateway.Usage)
		// OpenAI Responses API: auto-route based on group platform
		gateway.POST("/responses", func(c *gin.Context) {
			if rejectOpenAICompatibleMissingCapability(c, platformService, service.PlatformCapabilityResponses, "Responses API is not supported for this platform") {
				return
			}
			if isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityResponses) {
				h.OpenAIGateway.Responses(c)
				return
			}
			if isGroupGeminiCompatible(c, platformService) {
				openAIEndpointUnsupported(c, "Responses API is not supported for Gemini-compatible platforms")
				return
			}
			h.Gateway.Responses(c)
		})
		gateway.POST("/responses/*subpath", func(c *gin.Context) {
			if rejectOpenAICompatibleMissingCapability(c, platformService, service.PlatformCapabilityResponses, "Responses API is not supported for this platform") {
				return
			}
			if isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityResponses) {
				if isOpenAIResponsesPassthroughSubpath(c) {
					h.OpenAIGateway.ResponsesPassthrough(c)
					return
				}
				h.OpenAIGateway.Responses(c)
				return
			}
			if isGroupGeminiCompatible(c, platformService) {
				openAIEndpointUnsupported(c, "Responses API is not supported for Gemini-compatible platforms")
				return
			}
			h.Gateway.Responses(c)
		})
		gateway.GET("/responses", func(c *gin.Context) {
			if getGroupPlatform(c) == service.PlatformGrok {
				rejectGrokUnsupportedEndpoint(c, "Responses WebSocket API")
				return
			}
			openAICompatibleCapabilityOnly(platformService, service.PlatformCapabilityRealtime, "Realtime API is not supported for this platform", h.OpenAIGateway.ResponsesWebSocket)(c)
		})
		// OpenAI Chat Completions API: auto-route based on group platform
		gateway.POST("/chat/completions", func(c *gin.Context) {
			if rejectOpenAICompatibleMissingCapability(c, platformService, service.PlatformCapabilityChatCompletions, "Chat Completions API is not supported for this platform") {
				return
			}
			if getGroupPlatform(c) == service.PlatformGrok {
				rejectGrokUnsupportedEndpoint(c, "Chat Completions API")
				return
			}
			if isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityChatCompletions) {
				h.OpenAIGateway.ChatCompletions(c)
				return
			}
			if isGroupGeminiCompatible(c, platformService) {
				openAIEndpointUnsupported(c, "Chat Completions API is not supported for Gemini-compatible platforms")
				return
			}
			h.Gateway.ChatCompletions(c)
		})
		gateway.POST("/embeddings", func(c *gin.Context) {
			if getGroupPlatform(c) != service.PlatformOpenAI {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Embeddings API is not supported for this platform",
					},
				})
				return
			}
			h.OpenAIGateway.Embeddings(c)
		})
		gateway.POST("/images/generations", func(c *gin.Context) {
			if !isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityImages) {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Images API is not supported for this platform",
					},
				})
				return
			}
			h.OpenAIGateway.Images(c)
		})
		gateway.POST("/images/edits", func(c *gin.Context) {
			if !isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityImages) {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Images API is not supported for this platform",
					},
				})
				return
			}
			h.OpenAIGateway.Images(c)
		})
		registerXimoAIV1GatewayRoutes(gateway, ximoAIGatewayContext{
			platformService: platformService,
			handlers:        h,
		})
	}

	// Gemini 原生 API 兼容层（Gemini SDK/CLI 直连）
	gemini := r.Group("/v1beta")
	gemini.Use(bodyLimit)
	gemini.Use(clientRequestID)
	gemini.Use(opsErrorLogger)
	gemini.Use(endpointNorm)
	gemini.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	gemini.Use(requireGroupGoogle)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		gemini.GET("/operations/*operation", h.Gateway.GeminiV1BetaOperation)
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// OpenAI Responses API（不带v1前缀的别名）— auto-route based on group platform
	responsesHandler := func(c *gin.Context) {
		if rejectOpenAICompatibleMissingCapability(c, platformService, service.PlatformCapabilityResponses, "Responses API is not supported for this platform") {
			return
		}
		if isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityResponses) {
			if isOpenAIResponsesPassthroughSubpath(c) {
				h.OpenAIGateway.ResponsesPassthrough(c)
				return
			}
			h.OpenAIGateway.Responses(c)
			return
		}
		if isGroupGeminiCompatible(c, platformService) {
			openAIEndpointUnsupported(c, "Responses API is not supported for Gemini-compatible platforms")
			return
		}
		h.Gateway.Responses(c)
	}
	r.POST("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, responsesHandler)
	r.POST("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, responsesHandler)
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if getGroupPlatform(c) == service.PlatformGrok {
			rejectGrokUnsupportedEndpoint(c, "Responses WebSocket API")
			return
		}
		openAICompatibleCapabilityOnly(platformService, service.PlatformCapabilityRealtime, "Realtime API is not supported for this platform", h.OpenAIGateway.ResponsesWebSocket)(c)
	})
	codexDirect := r.Group("/backend-api/codex")
	codexDirect.Use(bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic)
	{
		codexDirect.POST("/responses", responsesHandler)
		codexDirect.POST("/responses/*subpath", responsesHandler)
		codexDirect.GET("/responses", func(c *gin.Context) {
			if getGroupPlatform(c) == service.PlatformGrok {
				rejectGrokUnsupportedEndpoint(c, "Responses WebSocket API")
				return
			}
			h.OpenAIGateway.ResponsesWebSocket(c)
		})
	}
	// OpenAI Chat Completions API（不带v1前缀的别名）— auto-route based on group platform
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if rejectOpenAICompatibleMissingCapability(c, platformService, service.PlatformCapabilityChatCompletions, "Chat Completions API is not supported for this platform") {
			return
		}
		if getGroupPlatform(c) == service.PlatformGrok {
			rejectGrokUnsupportedEndpoint(c, "Chat Completions API")
			return
		}
		if isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityChatCompletions) {
			h.OpenAIGateway.ChatCompletions(c)
			return
		}
		if isGroupGeminiCompatible(c, platformService) {
			openAIEndpointUnsupported(c, "Chat Completions API is not supported for Gemini-compatible platforms")
			return
		}
		h.Gateway.ChatCompletions(c)
	})
	r.POST("/embeddings", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if getGroupPlatform(c) != service.PlatformOpenAI {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Embeddings API is not supported for this platform",
				},
			})
			return
		}
		h.OpenAIGateway.Embeddings(c)
	})
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if !isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityImages) {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		h.OpenAIGateway.Images(c)
	})
	r.POST("/images/edits", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if !isGroupOpenAICompatibleWithCapability(c, platformService, service.PlatformCapabilityImages) {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		h.OpenAIGateway.Images(c)
	})
	registerXimoAIRootGatewayRoutes(r, ximoAIGatewayContext{
		bodyLimit:       bodyLimit,
		clientRequestID: clientRequestID,
		opsErrorLogger:  opsErrorLogger,
		endpointNorm:    endpointNorm,
		apiKeyAuth:      apiKeyAuth,
		requireGroup:    requireGroupAnthropic,
		platformService: platformService,
		handlers:        h,
	})

	// Antigravity 模型列表
	r.GET("/antigravity/models", gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 专用路由（仅使用 antigravity 账户，不混合调度）
	antigravityV1 := r.Group("/antigravity/v1")
	antigravityV1.Use(bodyLimit)
	antigravityV1.Use(clientRequestID)
	antigravityV1.Use(opsErrorLogger)
	antigravityV1.Use(endpointNorm)
	antigravityV1.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1.Use(gin.HandlerFunc(apiKeyAuth))
	antigravityV1.Use(requireGroupAnthropic)
	{
		antigravityV1.POST("/messages", h.Gateway.Messages)
		antigravityV1.POST("/messages/count_tokens", h.Gateway.CountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta")
	antigravityV1Beta.Use(bodyLimit)
	antigravityV1Beta.Use(clientRequestID)
	antigravityV1Beta.Use(opsErrorLogger)
	antigravityV1Beta.Use(endpointNorm)
	antigravityV1Beta.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1Beta.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	antigravityV1Beta.Use(requireGroupGoogle)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

}
