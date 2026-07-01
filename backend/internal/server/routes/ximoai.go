package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

func registerXimoAIAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
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
	handlers        *handler.Handlers
}

func registerXimoAIV1GatewayRoutes(gateway *gin.RouterGroup, ctx ximoAIGatewayContext) {
	h := ctx.handlers
	gateway.POST("/audio/speech", ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/transcriptions", ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/translations", ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/videos", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))
	gateway.GET("/videos", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))
	gateway.POST("/videos/*subpath", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	gateway.GET("/videos/*subpath", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
	gateway.DELETE("/videos/*subpath", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))
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
	r.POST("/audio/speech", withXimoAICommon(common, ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/audio/transcriptions", withXimoAICommon(common, ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/audio/translations", withXimoAICommon(common, ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/videos", withXimoAICommon(common, ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))...)
	r.GET("/videos", withXimoAICommon(common, ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))...)
	r.POST("/videos/*subpath", withXimoAICommon(common, ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
	r.GET("/videos/*subpath", withXimoAICommon(common, ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
	r.DELETE("/videos/*subpath", withXimoAICommon(common, ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
}

func withXimoAICommon(common []gin.HandlerFunc, last gin.HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(common)+1)
	handlers = append(handlers, common...)
	return append(handlers, last)
}

func getGroupPlatform(c *gin.Context) string {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

func openAIEndpointUnsupported(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"type":    "not_found_error",
			"message": message,
		},
	})
}
