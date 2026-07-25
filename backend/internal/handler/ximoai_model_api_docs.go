package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const modelAPIDocsMaxRequestBytes = 64 * 1024

type modelAPIDocsEditorResponse struct {
	AutomaticBinding  service.ModelAPIDocsBinding   `json:"automatic_binding"`
	AvailableProfiles []ModelAPIDocsEndpointProfile `json:"available_profiles"`
}

type modelAPIDocsResponse struct {
	Platform string                        `json:"platform"`
	Protocol string                        `json:"protocol"`
	Model    string                        `json:"model"`
	Source   string                        `json:"source"`
	Binding  service.ModelAPIDocsBinding   `json:"binding"`
	Profiles []ModelAPIDocsEndpointProfile `json:"profiles"`
	Editor   *modelAPIDocsEditorResponse   `json:"editor,omitempty"`
}

type saveModelAPIDocsRequest struct {
	Platform   string                                `json:"platform"`
	Protocol   string                                `json:"protocol"`
	Model      string                                `json:"model"`
	Categories []service.ModelAPIDocsCategoryBinding `json:"categories"`
}

type deleteModelAPIDocsRequest struct {
	Platform string `json:"platform"`
	Protocol string `json:"protocol"`
	Model    string `json:"model"`
}

func (h *AvailableChannelHandler) SaveModelPlazaDocs(c *gin.Context) {
	if h == nil || h.settingService == nil || h.platformService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_API_DOCS_UNAVAILABLE", "model API documentation settings are unavailable"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelAPIDocsMaxRequestBytes)
	var req saveModelAPIDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_API_DOCS_REQUEST_INVALID", "invalid model API documentation request"))
		return
	}
	binding := service.ModelAPIDocsBinding{
		Platform: req.Platform, Protocol: req.Protocol, Model: req.Model, Categories: req.Categories,
	}
	platform, err := h.platformService.GetBySlug(c.Request.Context(), binding.Platform)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_API_DOCS_PLATFORM_INVALID", "platform is unavailable"))
		return
	}
	if !strings.EqualFold(strings.TrimSpace(binding.Protocol), platform.Protocol) {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_API_DOCS_PROTOCOL_INVALID", "protocol does not match the platform"))
		return
	}
	if err := validateXimoAIModelAPIDocsBinding(binding, platform.Capabilities); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_API_DOCS_BINDING_INVALID", err.Error()))
		return
	}
	if err := h.settingService.SaveXimoAIModelAPIDocsBinding(c.Request.Context(), binding); err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("MODEL_API_DOCS_SAVE_FAILED", "failed to save model API documentation configuration").WithCause(err))
		return
	}
	response.Success(c, binding)
}

func (h *AvailableChannelHandler) DeleteModelPlazaDocs(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("MODEL_API_DOCS_UNAVAILABLE", "model API documentation settings are unavailable"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, modelAPIDocsMaxRequestBytes)
	var req deleteModelAPIDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_API_DOCS_REQUEST_INVALID", "invalid model API documentation request"))
		return
	}
	if strings.TrimSpace(req.Platform) == "" || strings.TrimSpace(req.Protocol) == "" || strings.TrimSpace(req.Model) == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MODEL_API_DOCS_REQUEST_INVALID", "platform, protocol, and model are required"))
		return
	}
	if err := h.settingService.DeleteXimoAIModelAPIDocsBinding(c.Request.Context(), req.Platform, req.Protocol, req.Model); err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("MODEL_API_DOCS_DELETE_FAILED", "failed to reset model API documentation configuration").WithCause(err))
		return
	}
	c.Status(http.StatusNoContent)
}

func applyModelAPIDocsTemplateValues(profiles []ModelAPIDocsEndpointProfile, model string) []ModelAPIDocsEndpointProfile {
	modelJSON, _ := json.Marshal(model)
	replacements := []struct {
		from string
		to   string
	}{
		{"{{MODEL_JSON}}", string(modelJSON)},
		{"{{MODEL_PATH}}", url.PathEscape(model)},
		{"{{MODEL}}", model},
	}
	for profileIndex := range profiles {
		for variantIndex := range profiles[profileIndex].Variants {
			for stepIndex := range profiles[profileIndex].Variants[variantIndex].Steps {
				step := &profiles[profileIndex].Variants[variantIndex].Steps[stepIndex]
				if strings.Contains(step.Path, "{model}") {
					step.Path = strings.ReplaceAll(step.Path, "{model}", url.PathEscape(model))
				}
				for _, replacement := range replacements {
					step.RequestExample = strings.ReplaceAll(step.RequestExample, replacement.from, replacement.to)
					step.ResponseExample = strings.ReplaceAll(step.ResponseExample, replacement.from, replacement.to)
				}
			}
		}
	}
	return profiles
}
