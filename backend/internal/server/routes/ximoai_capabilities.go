package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func isGroupOpenAICompatibleWithCapability(c *gin.Context, platformService *service.PlatformService, capability string) bool {
	if !isGroupOpenAICompatible(c, platformService) {
		return false
	}
	return groupPlatformSupportsCapability(c, platformService, capability)
}

func groupPlatformSupportsCapability(c *gin.Context, platformService *service.PlatformService, capability string) bool {
	if capability == "" {
		return true
	}
	platform := service.NormalizePlatformSlug(getGroupPlatform(c))
	if platform == "" {
		return false
	}
	if platform == service.PlatformOpenAI || platform == service.PlatformGemini {
		return service.NewPlatformService(nil).SupportsCapability(c.Request.Context(), platform, capability)
	}
	if platformService == nil {
		return false
	}
	return platformService.SupportsCapability(c.Request.Context(), platform, capability)
}

func openAICompatibleCapabilityOnly(platformService *service.PlatformService, capability string, message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isGroupOpenAICompatibleWithCapability(c, platformService, capability) {
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

func openAIVideosCapabilityOnly(platformService *service.PlatformService, message string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !(isGroupOpenAICompatible(c, platformService) || isGroupGeminiCompatible(c, platformService)) ||
			!groupPlatformSupportsCapability(c, platformService, service.PlatformCapabilityVideos) {
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

func rejectOpenAICompatibleMissingCapability(c *gin.Context, platformService *service.PlatformService, capability string, message string) bool {
	if !isGroupOpenAICompatible(c, platformService) {
		return false
	}
	if groupPlatformSupportsCapability(c, platformService, capability) {
		return false
	}
	openAIEndpointUnsupported(c, message)
	return true
}
