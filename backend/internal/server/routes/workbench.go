package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterWorkbenchRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	redisClient *redis.Client,
) {
	rateLimiter := appmiddleware.NewRateLimiter(redisClient)
	failClose := appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}
	workbench := v1.Group("/workbench")
	{
		workbench.POST("/sso-ticket/validate", rateLimiter.LimitWithOptions("workbench-sso-validate", 60, time.Minute, failClose), h.WorkbenchSSO.ValidateTicket)
		workbench.POST("/control-token/refresh", rateLimiter.LimitWithOptions("workbench-control-refresh", 30, time.Minute, failClose), h.WorkbenchSSO.RefreshControlToken)
		workbench.POST("/control-token/revoke", rateLimiter.LimitWithOptions("workbench-control-revoke", 30, time.Minute, failClose), h.WorkbenchSSO.RevokeControlToken)
	}

	authenticated := v1.Group("/workbench")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	{
		authenticated.POST("/sso-ticket", h.WorkbenchSSO.CreateTicket)
	}
}
