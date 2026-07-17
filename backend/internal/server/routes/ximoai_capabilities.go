package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func isGroupXimoAIAudioPlatform(c *gin.Context) bool {
	switch service.NormalizePlatformSlug(getGroupPlatform(c)) {
	case service.PlatformOpenAIAudio, service.PlatformKlingAudio:
		return true
	default:
		return false
	}
}

func isGrokVideoGatewayPlatform(c *gin.Context) bool {
	switch service.NormalizePlatformSlug(getGroupPlatform(c)) {
	case service.PlatformGrok, service.PlatformGrokVideo:
		return true
	default:
		return false
	}
}

func ximoAIAudioOnly(message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isGroupXimoAIAudioPlatform(c) {
			openAIEndpointUnsupported(c, message)
			return
		}
		next(c)
	}
}
