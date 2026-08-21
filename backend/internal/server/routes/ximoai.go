package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerXimoAIAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	admin.PUT("/model-plaza/metadata", h.AvailableChannel.SaveModelPlazaMetadata)
	admin.DELETE("/model-plaza/metadata", h.AvailableChannel.DeleteModelPlazaMetadata)
}

func registerXimoAIUserRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	authenticated.GET("/channels/model-plaza", h.AvailableChannel.ModelPlaza)
}
