package handler

import (
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const modelMetadataMaxRequestBytes = 64 * 1024

type saveModelMetadataRequest struct {
	Platform          string    `json:"platform"`
	Model             string    `json:"model"`
	Brand             *string   `json:"brand"`
	Types             *[]string `json:"types"`
	InvocationModes   *[]string `json:"invocation_modes"`
	ReasoningLevels   *[]string `json:"reasoning_levels"`
	ThinkingSupported *bool     `json:"thinking_supported"`
}

type deleteModelMetadataRequest struct {
	Platform string `json:"platform"`
	Model    string `json:"model"`
}

func (h *AvailableChannelHandler) SaveModelPlazaMetadata(c *gin.Context) {
	if h == nil || h.settingService == nil || h.platformService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_METADATA_UNAVAILABLE", "model metadata settings are unavailable"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelMetadataMaxRequestBytes)
	var req saveModelMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_METADATA_REQUEST_INVALID", "invalid model metadata request"))
		return
	}
	override := service.ModelMetadataOverride{
		Platform:          strings.ToLower(strings.TrimSpace(req.Platform)),
		Model:             strings.TrimSpace(req.Model),
		Brand:             req.Brand,
		Types:             req.Types,
		InvocationModes:   req.InvocationModes,
		ReasoningLevels:   req.ReasoningLevels,
		ThinkingSupported: req.ThinkingSupported,
	}
	if override.Platform == "" || override.Model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_METADATA_REQUEST_INVALID", "platform and model are required"))
		return
	}
	if _, err := h.platformService.GetBySlug(c.Request.Context(), override.Platform); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_METADATA_PLATFORM_INVALID", "platform is unavailable"))
		return
	}
	if override.Brand == nil && override.Types == nil && override.InvocationModes == nil && override.ReasoningLevels == nil && override.ThinkingSupported == nil {
		if err := h.settingService.DeleteXimoAIModelMetadataOverride(c.Request.Context(), override.Platform, override.Model); err != nil {
			response.ErrorFrom(c, infraerrors.InternalServer("MODEL_METADATA_SAVE_FAILED", "failed to restore automatic model metadata").WithCause(err))
			return
		}
		response.Success(c, modelMetadataOverrideResponse{})
		return
	}
	if err := h.settingService.SaveXimoAIModelMetadataOverride(c.Request.Context(), override); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_METADATA_INVALID", err.Error()))
		return
	}
	response.Success(c, modelMetadataOverrideResponse{
		Brand:             cloneStringPointer(override.Brand),
		Types:             cloneStringSlicePointer(override.Types),
		InvocationModes:   cloneStringSlicePointer(override.InvocationModes),
		ReasoningLevels:   cloneStringSlicePointer(override.ReasoningLevels),
		ThinkingSupported: cloneBoolPointer(override.ThinkingSupported),
	})
}

func (h *AvailableChannelHandler) DeleteModelPlazaMetadata(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_METADATA_UNAVAILABLE", "model metadata settings are unavailable"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelMetadataMaxRequestBytes)
	var req deleteModelMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_METADATA_REQUEST_INVALID", "invalid model metadata request"))
		return
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	model := strings.TrimSpace(req.Model)
	if platform == "" || model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_METADATA_REQUEST_INVALID", "platform and model are required"))
		return
	}
	if err := h.settingService.DeleteXimoAIModelMetadataOverride(c.Request.Context(), platform, model); err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("MODEL_METADATA_DELETE_FAILED", "failed to restore automatic model metadata").WithCause(err))
		return
	}
	response.Success(c, modelMetadataOverrideResponse{})
}
