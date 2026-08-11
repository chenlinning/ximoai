package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GetXimoAIHomeTabs returns the public XimoAI home workspace configuration.
func (h *SettingHandler) GetXimoAIHomeTabs(c *gin.Context) {
	tabs, err := h.settingService.GetXimoAIHomeTabs(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	enabled := make([]service.XimoAIHomeTab, 0, len(tabs))
	for _, tab := range tabs {
		if tab.Enabled {
			enabled = append(enabled, tab)
		}
	}
	response.Success(c, service.XimoAIPublicHomeTabs(enabled))
}
