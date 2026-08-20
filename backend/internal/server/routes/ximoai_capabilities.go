package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func ximoAIPlatformKind(c *gin.Context, gateway *handler.GatewayHandler) string {
	platform := service.NormalizePlatformSlug(getGroupPlatform(c))
	if gateway != nil {
		return gateway.XimoAIPlatformKind(c.Request.Context(), platform)
	}
	return service.XimoAIPlatformKindFromSlug(platform)
}

func isGrokVideoGatewayPlatform(c *gin.Context, gateway *handler.GatewayHandler) bool {
	platform := service.NormalizePlatformSlug(getGroupPlatform(c))
	return platform == service.PlatformGrok || ximoAIPlatformKind(c, gateway) == service.PlatformKindGrokVideo
}

func ximoAIAudioOnly(gateway *handler.GatewayHandler, allowKling bool, message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := service.NormalizePlatformSlug(getGroupPlatform(c))
		kind := ximoAIPlatformKind(c, gateway)
		allowed := kind == service.PlatformKindOpenAIAudio || (allowKling && kind == service.PlatformKindKlingAudio)
		if !allowed && gateway != nil {
			allowed = gateway.IsOpenAIAudioPlatform(c.Request.Context(), platform)
		}
		if !allowed {
			openAIEndpointUnsupported(c, message)
			return
		}
		next(c)
	}
}
