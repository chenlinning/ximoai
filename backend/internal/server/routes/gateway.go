package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 娉ㄥ唽 API 缃戝叧璺敱锛圕laude/OpenAI/Gemini 鍏煎锛
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

	// 鏈垎缁?Key 鎷︽埅涓棿浠讹紙鎸夊崗璁牸寮忓尯鍒嗛敊璇搷搴旓級
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)

	// API缃戝叧锛圕laude API鍏煎锛
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
			if isGroupOpenAICompatible(c, platformService) {
				h.OpenAIGateway.Messages(c)
				return
			}
			h.Gateway.Messages(c)
		})
		// /v1/messages/count_tokens: OpenAI groups get 404
		gateway.POST("/messages/count_tokens", func(c *gin.Context) {
			if isGroupOpenAICompatible(c, platformService) || isGroupGeminiCompatible(c, platformService) {
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
			if isGroupOpenAICompatible(c, platformService) {
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
			if isGroupOpenAICompatible(c, platformService) {
				h.OpenAIGateway.Responses(c)
				return
			}
			if isGroupGeminiCompatible(c, platformService) {
				openAIEndpointUnsupported(c, "Responses API is not supported for Gemini-compatible platforms")
				return
			}
			h.Gateway.Responses(c)
		})
		gateway.GET("/responses", officialOpenAIOnly(h.OpenAIGateway.ResponsesWebSocket))
		// OpenAI Chat Completions API: auto-route based on group platform
		gateway.POST("/chat/completions", func(c *gin.Context) {
			if isGroupOpenAICompatible(c, platformService) {
				h.OpenAIGateway.ChatCompletions(c)
				return
			}
			if isGroupGeminiCompatible(c, platformService) {
				openAIEndpointUnsupported(c, "Chat Completions API is not supported for Gemini-compatible platforms")
				return
			}
			h.Gateway.ChatCompletions(c)
		})
		gateway.POST("/images/generations", func(c *gin.Context) {
			if !isGroupOpenAICompatible(c, platformService) {
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
			if !isGroupOpenAICompatible(c, platformService) {
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
		gateway.POST("/videos", openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))
		gateway.GET("/videos", openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))
		gateway.POST("/videos/*subpath", openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
		gateway.GET("/videos/*subpath", openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
		gateway.DELETE("/videos/*subpath", openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	}

	// Gemini 鍘熺敓 API 鍏煎灞傦紙Gemini SDK/CLI 鐩磋繛锛
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
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// OpenAI Responses API锛堜笉甯1鍓嶇紑鐨勫埆鍚嶏級鈥?auto-route based on group platform
	responsesHandler := func(c *gin.Context) {
		if isGroupOpenAICompatible(c, platformService) {
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
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, officialOpenAIOnly(h.OpenAIGateway.ResponsesWebSocket))
	codexDirect := r.Group("/backend-api/codex")
	codexDirect.Use(bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic)
	{
		codexDirect.POST("/responses", officialOpenAIOnly(h.OpenAIGateway.Responses))
		codexDirect.POST("/responses/*subpath", officialOpenAIOnly(h.OpenAIGateway.Responses))
		codexDirect.GET("/responses", officialOpenAIOnly(h.OpenAIGateway.ResponsesWebSocket))
	}
	// OpenAI Chat Completions API锛堜笉甯1鍓嶇紑鐨勫埆鍚嶏級鈥?auto-route based on group platform
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGroupOpenAICompatible(c, platformService) {
			h.OpenAIGateway.ChatCompletions(c)
			return
		}
		if isGroupGeminiCompatible(c, platformService) {
			openAIEndpointUnsupported(c, "Chat Completions API is not supported for Gemini-compatible platforms")
			return
		}
		h.Gateway.ChatCompletions(c)
	})
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if !isGroupOpenAICompatible(c, platformService) {
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
		if !isGroupOpenAICompatible(c, platformService) {
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
	r.POST("/videos", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))
	r.GET("/videos", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))
	r.POST("/videos/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	r.GET("/videos/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	r.DELETE("/videos/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, openAICompatibleOnly(platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))

	// Antigravity 妯″瀷鍒楄〃
	r.GET("/antigravity/models", gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 涓撶敤璺敱锛堜粎浣跨敤 antigravity 璐︽埛锛屼笉娣峰悎璋冨害锛
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

// getGroupPlatform extracts the group platform from the API Key stored in context.
func getGroupPlatform(c *gin.Context) string {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

func isGroupOfficialOpenAI(c *gin.Context) bool {
	return service.NormalizePlatformSlug(getGroupPlatform(c)) == service.PlatformOpenAI
}

func isGroupOpenAICompatible(c *gin.Context, platformService *service.PlatformService) bool {
	platform := service.NormalizePlatformSlug(getGroupPlatform(c))
	if platform == "" {
		return false
	}
	if platform == service.PlatformOpenAI {
		return true
	}
	if platformService == nil {
		return false
	}
	return platformService.IsOpenAICompatible(c.Request.Context(), platform)
}

func isGroupGeminiCompatible(c *gin.Context, platformService *service.PlatformService) bool {
	platform := service.NormalizePlatformSlug(getGroupPlatform(c))
	if platform == "" {
		return false
	}
	if platform == service.PlatformGemini {
		return true
	}
	if platformService == nil {
		return false
	}
	return platformService.IsGeminiCompatible(c.Request.Context(), platform)
}

func openAIEndpointUnsupported(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"type":    "not_found_error",
			"message": message,
		},
	})
}

func officialOpenAIOnly(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isGroupOfficialOpenAI(c) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "This endpoint is only supported for official OpenAI groups",
				},
			})
			return
		}
		next(c)
	}
}

func openAICompatibleOnly(platformService *service.PlatformService, message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isGroupOpenAICompatible(c, platformService) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": message,
				},
			})
			return
		}
		next(c)
	}
}
