package routes

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func registerXimoAIAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	accounts := admin.Group("/accounts")
	{
		accounts.POST("/models/sync-upstream-preview", h.Admin.Account.SyncUpstreamModelsPreview)
	}

	platforms := admin.Group("/platforms")
	{
		platforms.GET("", h.Admin.Platform.List)
		platforms.POST("", h.Admin.Platform.Create)
		platforms.PUT("/:slug", h.Admin.Platform.Update)
		platforms.DELETE("/:slug", h.Admin.Platform.Delete)
	}
}

func registerXimoAIUserRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	authenticated.GET("/channels/model-plaza", h.AvailableChannel.ModelPlaza)

	platforms := authenticated.Group("/platforms")
	{
		platforms.GET("", h.Platform.ListPublic)
	}
}

type ximoAIGatewayContext struct {
	bodyLimit       gin.HandlerFunc
	clientRequestID gin.HandlerFunc
	opsErrorLogger  gin.HandlerFunc
	endpointNorm    gin.HandlerFunc
	apiKeyAuth      middleware.APIKeyAuthMiddleware
	requireGroup    gin.HandlerFunc
	platformService *service.PlatformService
	handlers        *handler.Handlers
}

func registerXimoAIV1GatewayRoutes(gateway *gin.RouterGroup, ctx ximoAIGatewayContext) {
	h := ctx.handlers
	gateway.GET("/responses/*subpath", openAICompatibleOnly(ctx.platformService, "Responses API is not supported for this platform", h.OpenAIGateway.ResponsesPassthrough))
	gateway.DELETE("/responses/*subpath", openAICompatibleOnly(ctx.platformService, "Responses API is not supported for this platform", h.OpenAIGateway.ResponsesPassthrough))
	gateway.POST("/audio/speech", openAICompatibleOnly(ctx.platformService, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/transcriptions", openAICompatibleOnly(ctx.platformService, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/translations", openAICompatibleOnly(ctx.platformService, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.GET("/realtime", openAICompatibleOnly(ctx.platformService, "Realtime API is not supported for this platform", h.OpenAIGateway.ResponsesWebSocket))
	gateway.POST("/videos", openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))
	gateway.GET("/videos", openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))
	gateway.POST("/videos/*subpath", openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	gateway.GET("/videos/*subpath", openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	gateway.DELETE("/videos/*subpath", openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
}

func registerXimoAIRootGatewayRoutes(r *gin.Engine, ctx ximoAIGatewayContext) {
	h := ctx.handlers
	common := []gin.HandlerFunc{
		ctx.bodyLimit,
		ctx.clientRequestID,
		ctx.opsErrorLogger,
		ctx.endpointNorm,
		gin.HandlerFunc(ctx.apiKeyAuth),
		ctx.requireGroup,
	}
	r.GET("/responses/*subpath", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Responses API is not supported for this platform", h.OpenAIGateway.ResponsesPassthrough))...)
	r.DELETE("/responses/*subpath", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Responses API is not supported for this platform", h.OpenAIGateway.ResponsesPassthrough))...)
	r.POST("/audio/speech", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/audio/transcriptions", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/audio/translations", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.GET("/realtime", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Realtime API is not supported for this platform", h.OpenAIGateway.ResponsesWebSocket))...)
	r.POST("/videos", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))...)
	r.GET("/videos", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))...)
	r.POST("/videos/*subpath", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
	r.GET("/videos/*subpath", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
	r.DELETE("/videos/*subpath", withXimoAICommon(common, openAICompatibleOnly(ctx.platformService, "Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
}

func withXimoAICommon(common []gin.HandlerFunc, last gin.HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(common)+1)
	handlers = append(handlers, common...)
	return append(handlers, last)
}

func isOpenAIResponsesPassthroughSubpath(c *gin.Context) bool {
	if c == nil {
		return false
	}
	subpath := strings.Trim(strings.TrimSpace(c.Param("subpath")), "/")
	if subpath == "" {
		return false
	}
	firstSegment := subpath
	if idx := strings.Index(firstSegment, "/"); idx >= 0 {
		firstSegment = firstSegment[:idx]
	}
	return !strings.EqualFold(firstSegment, "compact")
}

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
