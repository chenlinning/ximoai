package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ximoAIUserSupportedModel struct {
	Name              string                       `json:"name"`
	Platform          string                       `json:"platform"`
	Brand             string                       `json:"brand,omitempty"`
	MetadataEditor    *modelMetadataEditorResponse `json:"metadata_editor,omitempty"`
	Pricing           *userSupportedModelPricing   `json:"pricing"`
	Types             []string                     `json:"types"`
	InvocationModes   []string                     `json:"invocation_modes"`
	ReasoningLevels   []string                     `json:"reasoning_levels,omitempty"`
	ThinkingSupported bool                         `json:"thinking_supported,omitempty"`
	recognitionName   string
}

// ximoAIUserChannelPlatformSection 单渠道内某个平台的子视图：用户可见的分组 + 该平台
// 支持的模型。按 platform 聚合后让前端可以把渠道名作为 row-group 一次渲染，
// 后面的平台行按 sections 顺序铺开。
type ximoAIUserChannelPlatformSection struct {
	Platform        string                     `json:"platform"`
	DisplayName     string                     `json:"display_name"`
	Color           string                     `json:"color"`
	Protocol        string                     `json:"protocol"`
	Groups          []userAvailableGroup       `json:"groups"`
	SupportedModels []ximoAIUserSupportedModel `json:"supported_models"`
}

// ximoAIUserAvailableChannel 用户可见的渠道条目（白名单字段）。
//
// 每个渠道聚合为一条记录，内嵌 platforms 子数组：每个 section 对应一个平台，
// 包含该平台的 groups 和 supported_models。
type ximoAIUserAvailableChannel struct {
	Name        string                             `json:"name"`
	Description string                             `json:"description"`
	Platforms   []ximoAIUserChannelPlatformSection `json:"platforms"`
}

// ModelPlaza lists user-visible model pricing without checking the optional
// Available Channels feature flag.
// GET /api/v1/channels/model-plaza
func (h *AvailableChannelHandler) ModelPlaza(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	out, err := h.listXimoAIVisibleChannels(c, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	role, _ := middleware.GetUserRoleFromContext(c)
	if err := h.enrichModelPlazaCatalog(c.Request.Context(), subject.UserID, out, role == service.RoleAdmin); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, out)
}

func (h *AvailableChannelHandler) listXimoAIVisibleChannels(c *gin.Context, userID int64) ([]ximoAIUserAvailableChannel, error) {
	userGroups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	allowedGroupIDs := make(map[int64]struct{}, len(userGroups))
	for i := range userGroups {
		allowedGroupIDs[userGroups[i].ID] = struct{}{}
	}

	channels, err := h.channelService.ListAvailableChannelPricing(c.Request.Context())
	if err != nil {
		return nil, err
	}

	acceptLanguage := c.GetHeader("Accept-Language")
	out := make([]ximoAIUserAvailableChannel, 0, len(channels))
	for _, ch := range channels {
		if ch.Status != service.StatusActive {
			continue
		}
		visibleGroups := filterUserVisibleGroups(ch.Groups, allowedGroupIDs)
		if len(visibleGroups) == 0 {
			continue
		}
		localizeUserAvailableGroups(visibleGroups, acceptLanguage)
		sections := buildXimoAIPlatformSections(ch, visibleGroups)
		if len(sections) == 0 {
			continue
		}
		out = append(out, ximoAIUserAvailableChannel{
			Name:        ch.Name,
			Description: ch.Description,
			Platforms:   sections,
		})
	}

	return out, nil
}

func buildXimoAIPlatformSections(ch service.XimoAIAvailableChannel, groups []userAvailableGroup) []ximoAIUserChannelPlatformSection {
	sections := buildPlatformSections(ch.AvailableChannel, groups)
	result := make([]ximoAIUserChannelPlatformSection, 0, len(sections))
	for _, section := range sections {
		models := make([]ximoAIUserSupportedModel, 0, len(section.SupportedModels))
		for _, model := range section.SupportedModels {
			models = append(models, ximoAIUserSupportedModel{
				Name: model.Name, Platform: model.Platform, Pricing: model.Pricing,
				recognitionName: ch.RecognitionName(model.Platform, model.Name),
			})
		}
		result = append(result, ximoAIUserChannelPlatformSection{
			Platform: section.Platform, Groups: section.Groups, SupportedModels: models,
		})
	}
	return result
}
