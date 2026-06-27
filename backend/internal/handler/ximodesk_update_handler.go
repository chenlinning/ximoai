package handler

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const ximoDeskUpdateRequestBodyLimit = 16 << 10
const ximoDeskPackageUploadBodyLimit = 1024 << 20

type XimoDeskUpdateHandler struct {
	settingService *service.SettingService
}

func NewXimoDeskUpdateHandler(settingService *service.SettingService) *XimoDeskUpdateHandler {
	return &XimoDeskUpdateHandler{settingService: settingService}
}

func (h *XimoDeskUpdateHandler) LatestApp(c *gin.Context) {
	if h == nil || h.settingService == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ximoDeskUpdateRequestBodyLimit)
	var req service.XimoDeskVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body is too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	update, ok, err := h.settingService.ResolveXimoAppUpdate(c.Request.Context(), c.Param("appKey"), req)
	if err != nil || !ok || update == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, update)
}

func (h *XimoDeskUpdateHandler) AdminGet(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "XimoAPP update service is not configured")
		return
	}
	cfg, err := h.settingService.GetXimoDeskUpdateConfig(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, cfg)
}

func (h *XimoDeskUpdateHandler) AdminUpdate(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "XimoAPP update service is not configured")
		return
	}
	var cfg service.XimoDeskUpdateConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.settingService.SaveXimoDeskUpdateConfig(c.Request.Context(), &cfg); response.ErrorFrom(c, err) {
		return
	}
	saved, err := h.settingService.GetXimoDeskUpdateConfig(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, saved)
}

func (h *XimoDeskUpdateHandler) AdminUploadPackage(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "XimoAPP update service is not configured")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ximoDeskPackageUploadBodyLimit)
	if err := c.Request.ParseMultipartForm(ximoDeskPackageUploadBodyLimit); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.BadRequest(c, "Package file is too large")
			return
		}
		response.BadRequest(c, "Invalid multipart package upload: "+err.Error())
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Package file is required")
		return
	}
	defer func() { _ = file.Close() }()

	release, cfg, err := h.settingService.SaveXimoDeskPackageRelease(c.Request.Context(), service.XimoDeskPackageUpload{
		AppKey:                  c.PostForm("app_key"),
		Channel:                 c.PostForm("channel"),
		OS:                      c.PostForm("os"),
		Arch:                    c.PostForm("arch"),
		Locale:                  c.PostForm("locale"),
		Version:                 c.PostForm("version"),
		MinSupportedVersion:     c.PostForm("min_supported_version"),
		MinSupportedVersionCode: parseXimoDeskInt64(c.PostForm("min_supported_version_code")),
		PackageType:             c.PostForm("package_type"),
		OriginalName:            header.Filename,
		Notes:                   c.PostForm("notes"),
		Force:                   parseXimoDeskBool(c.PostForm("force"), false),
		Enabled:                 parseXimoDeskBool(c.PostForm("enabled"), true),
		Reader:                  file,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{
		"release": release,
		"config":  cfg,
	})
}

func (h *XimoDeskUpdateHandler) AdminDeleteRelease(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "XimoAPP update service is not configured")
		return
	}
	cfg, err := h.settingService.DeleteXimoDeskUpdateRelease(c.Request.Context(), c.Param("id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, cfg)
}

func (h *XimoDeskUpdateHandler) DownloadCenter(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "XimoAPP download center service is not configured")
		return
	}
	center, err := h.settingService.GetXimoAppDownloadCenter(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, center)
}

func (h *XimoDeskUpdateHandler) DownloadPackage(c *gin.Context) {
	filePath, err := service.XimoDeskPackageLookupPath(c.Param("file"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(filePath)
}

func parseXimoDeskInt64(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseXimoDeskBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
