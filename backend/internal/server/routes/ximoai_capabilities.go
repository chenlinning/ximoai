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

func isGroupXimoAIVideoPlatform(c *gin.Context) bool {
	return service.NormalizePlatformSlug(getGroupPlatform(c)) == service.PlatformGrokVideo
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

func ximoAIVideoOnly(message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isGroupXimoAIVideoPlatform(c) {
			openAIEndpointUnsupported(c, message)
			return
		}
		next(c)
	}
}
