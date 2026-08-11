package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const ximoAIPublicAssetPrefix = "/api/v1/settings/assets"

type XimoAIPublicAsset struct {
	Content     []byte
	ContentType string
	Filename    string
	ETag        string
}

func XimoAIPublicHomeTabs(tabs []XimoAIHomeTab) []XimoAIHomeTab {
	publicTabs := make([]XimoAIHomeTab, len(tabs))
	copy(publicTabs, tabs)
	for index := range publicTabs {
		asset, ok := decodeXimoAIDataAsset(publicTabs[index].CoverURL)
		if !ok || !isXimoAIHomeCoverContentType(asset.ContentType) {
			continue
		}
		publicTabs[index].CoverURL = fmt.Sprintf(
			"%s/home-tabs/%s/cover/%s",
			ximoAIPublicAssetPrefix,
			publicTabs[index].ID,
			asset.Filename,
		)
	}
	return publicTabs
}

func XimoAIHomeTabsUsePublicHTMLAssets(tabs []XimoAIHomeTab) bool {
	for _, tab := range tabs {
		if !tab.Enabled {
			continue
		}
		asset, ok := decodeXimoAIDataAsset(tab.CoverURL)
		if ok && (asset.ContentType == "text/html" || asset.ContentType == "application/xhtml+xml") {
			return true
		}
	}
	return false
}

func (s *SettingService) GetXimoAIPublicHomeCover(ctx context.Context, tabID, filename string) (*XimoAIPublicAsset, error) {
	tabs, err := s.GetXimoAIHomeTabs(ctx)
	if err != nil {
		return nil, err
	}
	for _, tab := range tabs {
		if tab.ID != tabID || !tab.Enabled {
			continue
		}
		asset, ok := decodeXimoAIDataAsset(tab.CoverURL)
		if !ok || !isXimoAIHomeCoverContentType(asset.ContentType) || asset.Filename != filename {
			break
		}
		return asset, nil
	}
	return nil, ximoAIPublicAssetNotFound()
}

func decodeXimoAIDataAsset(raw string) (*XimoAIPublicAsset, bool) {
	raw = strings.TrimSpace(raw)
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return nil, false
	}
	header := strings.ToLower(raw[:comma])
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return nil, false
	}
	contentType := strings.TrimSpace(strings.SplitN(header[5:], ";", 2)[0])
	if contentType == "" {
		return nil, false
	}
	content, err := base64.StdEncoding.DecodeString(raw[comma+1:])
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	return &XimoAIPublicAsset{
		Content:     content,
		ContentType: contentType,
		Filename:    digest + ximoAIPublicAssetExtension(contentType),
		ETag:        `"` + digest + `"`,
	}, true
}

func ximoAIPublicAssetExtension(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "video/webm":
		return ".webm"
	case "video/ogg":
		return ".ogg"
	case "video/quicktime":
		return ".mov"
	case "text/html":
		return ".html"
	case "application/xhtml+xml":
		return ".xhtml"
	default:
		if strings.HasPrefix(contentType, "video/") {
			return ".mp4"
		}
		return ".img"
	}
}

func isXimoAIHomeCoverContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "video/") ||
		contentType == "text/html" ||
		contentType == "application/xhtml+xml"
}

func ximoAIPublicAssetNotFound() error {
	return infraerrors.NotFound("XIMOAI_PUBLIC_ASSET_NOT_FOUND", "public asset not found")
}
