package handler

import (
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type saveModelBrandRequest struct {
	Platform string `json:"platform"`
	Model    string `json:"model"`
	Brand    string `json:"brand"`
}

type deleteModelBrandRequest struct {
	Platform string `json:"platform"`
	Model    string `json:"model"`
}

func (h *AvailableChannelHandler) SaveModelPlazaBrand(c *gin.Context) {
	if h == nil || h.settingService == nil || h.platformService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_BRAND_UNAVAILABLE", "model brand settings are unavailable"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelAPIDocsMaxRequestBytes)
	var req saveModelBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_BRAND_REQUEST_INVALID", "invalid model brand request"))
		return
	}
	override := service.ModelBrandOverride{
		Platform: strings.ToLower(strings.TrimSpace(req.Platform)),
		Model:    strings.TrimSpace(req.Model),
		Brand:    strings.TrimSpace(req.Brand),
	}
	if override.Platform == "" || override.Model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_BRAND_REQUEST_INVALID", "platform and model are required"))
		return
	}
	if err := service.ValidateXimoAIModelBrand(override.Brand); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_BRAND_INVALID", err.Error()))
		return
	}
	if _, err := h.platformService.GetBySlug(c.Request.Context(), override.Platform); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_BRAND_PLATFORM_INVALID", "platform is unavailable"))
		return
	}
	if err := h.settingService.SaveXimoAIModelBrandOverride(c.Request.Context(), override); err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("MODEL_BRAND_SAVE_FAILED", "failed to save model brand configuration").WithCause(err))
		return
	}
	response.Success(c, buildXimoAIModelBrandDetails(override.Model, &override, true))
}

func (h *AvailableChannelHandler) DeleteModelPlazaBrand(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_BRAND_UNAVAILABLE", "model brand settings are unavailable"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelAPIDocsMaxRequestBytes)
	var req deleteModelBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_BRAND_REQUEST_INVALID", "invalid model brand request"))
		return
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	model := strings.TrimSpace(req.Model)
	if platform == "" || model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_BRAND_REQUEST_INVALID", "platform and model are required"))
		return
	}
	if err := h.settingService.DeleteXimoAIModelBrandOverride(c.Request.Context(), platform, model); err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("MODEL_BRAND_DELETE_FAILED", "failed to restore automatic model brand").WithCause(err))
		return
	}
	response.Success(c, buildXimoAIModelBrandDetails(model, nil, true))
}
