package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type WorkbenchModelAccessHandler struct {
	modelAccessService *service.WorkbenchModelAccessService
}

type workbenchModelAccessRequest struct {
	UserID string `json:"userId" binding:"required"`
}

func NewWorkbenchModelAccessHandler(modelAccessService *service.WorkbenchModelAccessService) *WorkbenchModelAccessHandler {
	return &WorkbenchModelAccessHandler{modelAccessService: modelAccessService}
}

func (h *WorkbenchModelAccessHandler) Get(c *gin.Context) {
	if !h.authorizeInternal(c) {
		response.Unauthorized(c, "Invalid Workbench model access secret")
		return
	}
	var req workbenchModelAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(req.UserID), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid userId")
		return
	}
	modelConfig, err := h.modelAccessService.GetUserModelConfig(c.Request.Context(), userID)
	if response.ErrorFrom(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"modelConfig": modelConfig})
}

func (h *WorkbenchModelAccessHandler) authorizeInternal(c *gin.Context) bool {
	if h == nil || h.modelAccessService == nil {
		return false
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	return service.ConstantTimeBearerEqual(
		strings.TrimSpace(strings.TrimPrefix(header, prefix)),
		h.modelAccessService.InternalSecret(),
	)
}
