package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type updateXimoAIHomeTabsRequest struct {
	Tabs []service.XimoAIHomeTab `json:"tabs"`
}

func (h *SettingHandler) GetXimoAIHomeTabs(c *gin.Context) {
	tabs, err := h.settingService.GetXimoAIHomeTabs(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tabs)
}

func (h *SettingHandler) UpdateXimoAIHomeTabs(c *gin.Context) {
	var request updateXimoAIHomeTabsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid home tabs request")
		return
	}
	tabs, err := h.settingService.UpdateXimoAIHomeTabs(c.Request.Context(), request.Tabs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tabs)
}
