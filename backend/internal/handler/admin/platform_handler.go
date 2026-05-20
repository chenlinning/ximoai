package admin

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type PlatformHandler struct {
	platformService *service.PlatformService
}

func NewPlatformHandler(platformService *service.PlatformService) *PlatformHandler {
	return &PlatformHandler{platformService: platformService}
}

type platformRequest struct {
	Slug         string   `json:"slug"`
	DisplayName  string   `json:"display_name"`
	Protocol     string   `json:"protocol"`
	BaseURL      string   `json:"base_url"`
	AuthModes    []string `json:"auth_modes"`
	Capabilities []string `json:"capabilities"`
	Color        string   `json:"color"`
	Enabled      *bool    `json:"enabled"`
}

type platformResponse struct {
	Slug         string   `json:"slug"`
	DisplayName  string   `json:"display_name"`
	Protocol     string   `json:"protocol"`
	BaseURL      string   `json:"base_url"`
	AuthModes    []string `json:"auth_modes"`
	Capabilities []string `json:"capabilities"`
	Color        string   `json:"color"`
	Enabled      bool     `json:"enabled"`
	Builtin      bool     `json:"builtin"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

func (h *PlatformHandler) List(c *gin.Context) {
	includeDisabled := strings.EqualFold(c.Query("include_disabled"), "true")
	platforms, err := h.platformService.List(c.Request.Context(), includeDisabled)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]platformResponse, 0, len(platforms))
	for i := range platforms {
		out = append(out, platformToResponse(platforms[i]))
	}
	response.Success(c, out)
}

func (h *PlatformHandler) ListPublic(c *gin.Context) {
	platforms, err := h.platformService.List(c.Request.Context(), false)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]platformResponse, 0, len(platforms))
	for i := range platforms {
		item := platformToResponse(platforms[i])
		item.BaseURL = ""
		item.AuthModes = []string{}
		out = append(out, item)
	}
	response.Success(c, out)
}

func (h *PlatformHandler) Create(c *gin.Context) {
	var req platformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	platform := requestToPlatform(req)
	if req.Enabled != nil {
		platform.Enabled = *req.Enabled
	} else {
		platform.Enabled = true
	}
	created, err := h.platformService.Create(c.Request.Context(), platform)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, platformToResponse(*created))
}

func (h *PlatformHandler) Update(c *gin.Context) {
	var req platformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	platform := requestToPlatform(req)
	if req.Enabled != nil {
		platform.Enabled = *req.Enabled
	} else {
		current, err := h.platformService.GetBySlug(c.Request.Context(), c.Param("slug"))
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		platform.Enabled = current.Enabled
	}
	updated, err := h.platformService.Update(c.Request.Context(), c.Param("slug"), platform)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, platformToResponse(*updated))
}

func (h *PlatformHandler) Delete(c *gin.Context) {
	if err := h.platformService.Delete(c.Request.Context(), c.Param("slug")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Platform deleted successfully"})
}

func requestToPlatform(req platformRequest) service.Platform {
	return service.Platform{
		Slug:         req.Slug,
		DisplayName:  req.DisplayName,
		Protocol:     req.Protocol,
		BaseURL:      req.BaseURL,
		AuthModes:    req.AuthModes,
		Capabilities: req.Capabilities,
		Color:        req.Color,
	}
}

func platformToResponse(p service.Platform) platformResponse {
	return platformResponse{
		Slug:         p.Slug,
		DisplayName:  p.DisplayName,
		Protocol:     p.Protocol,
		BaseURL:      p.BaseURL,
		AuthModes:    p.AuthModes,
		Capabilities: p.Capabilities,
		Color:        p.Color,
		Enabled:      p.Enabled,
		Builtin:      p.Builtin,
		CreatedAt:    p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
