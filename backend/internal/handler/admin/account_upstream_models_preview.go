package admin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SyncUpstreamModelsPreviewRequest struct {
	Platform    string         `json:"platform" binding:"required"`
	Type        string         `json:"type" binding:"required,oneof=apikey upstream"`
	Credentials map[string]any `json:"credentials" binding:"required"`
}

// SyncUpstreamModelsPreview syncs live supported models before the account is created.
// POST /api/v1/admin/accounts/models/sync-upstream-preview
func (h *AccountHandler) SyncUpstreamModelsPreview(c *gin.Context) {
	if h.accountTestService == nil {
		response.InternalError(c, "Account test service is not configured")
		return
	}

	var req SyncUpstreamModelsPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account := &service.Account{
		ID:          0,
		Name:        "preview",
		Platform:    service.NormalizePlatformSlug(req.Platform),
		Type:        req.Type,
		Credentials: req.Credentials,
		Status:      service.StatusActive,
	}
	models, err := h.accountTestService.FetchUpstreamSupportedModels(c.Request.Context(), account)
	if err != nil {
		handleSyncUpstreamModelsPreviewError(c, err, "platform", account.Platform)
		return
	}

	response.Success(c, gin.H{"models": models})
}

func handleSyncUpstreamModelsPreviewError(c *gin.Context, err error, logKey string, logValue any) {
	var syncErr *service.UpstreamModelSyncError
	if errors.As(err, &syncErr) {
		switch syncErr.Kind {
		case service.UpstreamModelSyncErrorConfiguration, service.UpstreamModelSyncErrorUnsupported:
			response.BadRequest(c, syncErr.SafeMessage())
		default:
			slog.Warn("sync_upstream_models_preview_failed", logKey, logValue, "kind", syncErr.Kind)
			response.Error(c, http.StatusBadGateway, syncErr.SafeMessage())
		}
		return
	}

	slog.Warn("sync_upstream_models_preview_failed", logKey, logValue)
	response.Error(c, http.StatusBadGateway, "Failed to sync upstream models from upstream")
}
