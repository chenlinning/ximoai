package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

func registerXimoAIAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	admin.PUT("/model-plaza/docs", h.AvailableChannel.SaveModelPlazaDocs)
	admin.DELETE("/model-plaza/docs", h.AvailableChannel.DeleteModelPlazaDocs)
	admin.PUT("/model-plaza/brand", h.AvailableChannel.SaveModelPlazaBrand)
	admin.DELETE("/model-plaza/brand", h.AvailableChannel.DeleteModelPlazaBrand)

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

	videoCollector := authenticated.Group("/video-collector")
	{
		videoCollector.POST("/parse", h.VideoCollector.Parse)
		videoCollector.POST("/tasks", h.VideoCollector.Start)
		videoCollector.GET("/tasks/:id", h.VideoCollector.GetTask)
		videoCollector.DELETE("/tasks/:id", h.VideoCollector.Cancel)
		videoCollector.GET("/tasks/:id/download", h.VideoCollector.Download)
	}

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
	gateway.POST("/volcengine/images/generations", h.Gateway.VolcengineAgentPlanImages)
	gateway.POST("/volcengine/audio/tts/unidirectional", h.Gateway.VolcengineAgentPlanTTSUnidirectional)
	gateway.GET("/volcengine/audio/tts/unidirectional/stream", h.Gateway.VolcengineAgentPlanTTSUnidirectionalStream)
	gateway.GET("/volcengine/audio/tts/bidirection", h.Gateway.VolcengineAgentPlanTTSBidirection)
	gateway.GET("/volcengine/audio/asr/bigmodel_async", h.Gateway.VolcengineAgentPlanASRBigmodelAsync)
	gateway.GET("/volcengine/audio/asr/bigmodel_nostream", h.Gateway.VolcengineAgentPlanASRBigmodelNostream)
	gateway.POST("/audio/speech", ximoAIAudioOnly(h.Gateway, true, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/transcriptions", ximoAIAudioOnly(h.Gateway, false, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/translations", ximoAIAudioOnly(h.Gateway, false, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/videos", func(c *gin.Context) { openAIEndpointUnsupported(c, "Video endpoint not found") })
	gateway.GET("/videos", func(c *gin.Context) { openAIEndpointUnsupported(c, "Video endpoint not found") })
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
	r.POST("/audio/speech", withXimoAICommon(common, ximoAIAudioOnly(h.Gateway, true, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/audio/transcriptions", withXimoAICommon(common, ximoAIAudioOnly(h.Gateway, false, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/audio/translations", withXimoAICommon(common, ximoAIAudioOnly(h.Gateway, false, "Audio API is not supported for this platform", h.OpenAIGateway.Audio))...)
	r.POST("/videos", withXimoAICommon(common, func(c *gin.Context) { openAIEndpointUnsupported(c, "Video endpoint not found") })...)
	r.GET("/videos", withXimoAICommon(common, func(c *gin.Context) { openAIEndpointUnsupported(c, "Video endpoint not found") })...)
}

func withXimoAICommon(common []gin.HandlerFunc, last gin.HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(common)+1)
	handlers = append(handlers, common...)
	return append(handlers, last)
}

func openAIEndpointUnsupported(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"type":    "not_found_error",
			"message": message,
		},
	})
}
