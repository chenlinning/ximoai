package routes

import (
	"net/http"
	"strings"

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
	grokVideoCreate gin.HandlerFunc
	grokVideoStatus gin.HandlerFunc
}

func registerXimoAIV1GatewayRoutes(gateway *gin.RouterGroup, ctx ximoAIGatewayContext) {
	h := ctx.handlers
	gateway.POST("/audio/speech", ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/transcriptions", ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/audio/translations", ximoAIAudioOnly("Audio API is not supported for this platform", h.OpenAIGateway.Audio))
	gateway.POST("/videos", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosCreate))
	gateway.GET("/videos", ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosRetrieve))
	gateway.POST("/videos/*subpath", ximoAIVideoPostSubpath(ctx.grokVideoCreate, h.OpenAIGateway.VideosSubpath))
	gateway.GET("/videos/*subpath", ximoAIVideoGetSubpath(ctx.grokVideoStatus, h.OpenAIGateway.VideosSubpath))
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
	r.POST("/videos/*subpath", withXimoAICommon(common, ximoAIVideoPostSubpath(ctx.grokVideoCreate, h.OpenAIGateway.VideosSubpath))...)
	r.GET("/videos/*subpath", withXimoAICommon(common, ximoAIVideoGetSubpath(ctx.grokVideoStatus, h.OpenAIGateway.VideosSubpath))...)
	r.DELETE("/videos/*subpath", withXimoAICommon(common, ximoAIVideoOnly("Videos API is not supported for this platform", h.OpenAIGateway.VideosSubpath))...)
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

func ximoAIVideoPostSubpath(grokVideoCreate, ximoVideo gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		subpath := strings.Trim(strings.TrimSpace(c.Param("subpath")), "/")
		if strings.EqualFold(subpath, "generations") && !isGroupXimoAIVideoPlatform(c) {
			grokVideoCreate(c)
			return
		}
		if !isGroupXimoAIVideoPlatform(c) {
			openAIEndpointUnsupported(c, "Videos API is not supported for this platform")
			return
		}
		ximoVideo(c)
	}
}

func ximoAIVideoGetSubpath(grokVideoStatus, ximoVideo gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		subpath := strings.Trim(strings.TrimSpace(c.Param("subpath")), "/")
		if !isGroupXimoAIVideoPlatform(c) {
			setGinParam(c, "request_id", strings.Trim(subpath, "/"))
			grokVideoStatus(c)
			return
		}
		ximoVideo(c)
	}
}

func setGinParam(c *gin.Context, key, value string) {
	for i := range c.Params {
		if c.Params[i].Key == key {
			c.Params[i].Value = value
			return
		}
	}
	c.Params = append(c.Params, gin.Param{Key: key, Value: value})
}
