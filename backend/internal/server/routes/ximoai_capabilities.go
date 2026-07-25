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
	return service.XimoAIPlatformKindFromLegacySlug(platform)
}

func isGrokVideoGatewayPlatform(c *gin.Context, gateway *handler.GatewayHandler) bool {
	platform := service.NormalizePlatformSlug(getGroupPlatform(c))
	return platform == service.PlatformGrok || ximoAIPlatformKind(c, gateway) == service.PlatformKindGrokVideo
}

func ximoAIAudioOnly(gateway *handler.GatewayHandler, allowKling bool, message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := ximoAIPlatformKind(c, gateway)
		if kind != service.PlatformKindOpenAIAudio && (!allowKling || kind != service.PlatformKindKlingAudio) {
			openAIEndpointUnsupported(c, message)
			return
		}
		next(c)
	}
}
