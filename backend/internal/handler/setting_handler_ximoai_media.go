package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const ximoAIPublicAssetCSP = "sandbox; default-src 'none'; style-src 'unsafe-inline' https:; img-src data: https:; media-src data: https:; font-src data: https:"

func (h *SettingHandler) GetXimoAIPublicHomeCover(c *gin.Context) {
	asset, err := h.settingService.GetXimoAIPublicHomeCover(
		c.Request.Context(),
		c.Param("id"),
		c.Param("filename"),
	)
	if response.ErrorFrom(c, err) {
		return
	}
	serveXimoAIPublicAsset(c, asset)
}

func serveXimoAIPublicAsset(c *gin.Context, asset *service.XimoAIPublicAsset) {
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Header("Content-Security-Policy", ximoAIPublicAssetCSP)
	c.Header("ETag", asset.ETag)
	if asset.ContentType == "text/html" || asset.ContentType == "application/xhtml+xml" {
		c.Header("X-Frame-Options", "SAMEORIGIN")
	}
	if ximoAIPublicAssetETagMatches(c.GetHeader("If-None-Match"), asset.ETag) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, asset.ContentType, asset.Content)
}

func ximoAIPublicAssetETagMatches(header, etag string) bool {
	target := strings.TrimPrefix(strings.TrimSpace(etag), "W/")
	for _, raw := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(raw)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == target {
			return true
		}
	}
	return false
}
