package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterDesktopRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, redisClient *redis.Client) {
	rateLimiter := appmiddleware.NewRateLimiter(redisClient)
	failClose := appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}
	limit := rateLimiter.LimitWithOptions("desktop-session", 30, time.Minute, failClose)

	desktop := v1.Group("/desktop")
	{
		desktop.POST("/token", limit, h.DesktopSession.Token)
		desktop.POST("/sso-ticket", limit, h.DesktopSession.SSOTicket)
		desktop.POST("/sso-broker-credential", limit, h.DesktopSession.SSOBrokerCredential)
		desktop.DELETE("/session", limit, h.DesktopSession.RevokeSession)
		desktop.POST("/revoke", limit, h.DesktopSession.RevokeRefresh)
	}

	authorized := v1.Group("/desktop")
	authorized.Use(gin.HandlerFunc(jwtAuth), limit)
	{
		authorized.POST("/authorize", h.DesktopSession.Authorize)
	}
}
