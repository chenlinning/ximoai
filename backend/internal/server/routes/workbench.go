package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterWorkbenchRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
) {
	workbench := v1.Group("/workbench")
	{
		workbench.POST("/sso-ticket/validate", h.WorkbenchSSO.ValidateTicket)
	}

	authenticated := v1.Group("/workbench")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	{
		authenticated.POST("/sso-ticket", h.WorkbenchSSO.CreateTicket)
	}
}
