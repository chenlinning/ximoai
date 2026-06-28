package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterXimoDeskRoutes(r *gin.Engine, settingService *service.SettingService) {
	if r == nil || settingService == nil {
		return
	}
	h := handler.NewXimoDeskUpdateHandler(settingService)
	r.POST("/api/ximoapp/:appKey/version/latest", h.LatestApp)
	r.GET("/downloads/ximoapp/:file", h.DownloadPackage)
}

func RegisterXimoAppDownloadRoutes(v1 *gin.RouterGroup, jwtAuth middleware.JWTAuthMiddleware, settingService *service.SettingService) {
	if v1 == nil || settingService == nil {
		return
	}
	h := handler.NewXimoDeskUpdateHandler(settingService)
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	authenticated.GET("/ximoapp/download-center", h.DownloadCenter)
}

func registerXimoDeskAdminRoutes(admin *gin.RouterGroup, settingService *service.SettingService) {
	if admin == nil || settingService == nil {
		return
	}
	h := handler.NewXimoDeskUpdateHandler(settingService)
	ximoapp := admin.Group("/ximoapp")
	{
		ximoapp.GET("/update", h.AdminGet)
		ximoapp.PUT("/update", h.AdminUpdate)
		ximoapp.POST("/update/packages", h.AdminUploadPackage)
		ximoapp.DELETE("/update/releases/:id", h.AdminDeleteRelease)
		ximoapp.DELETE("/update/apps/:appKey", h.AdminDeleteApp)
	}
}
