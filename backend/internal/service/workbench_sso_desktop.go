package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *WorkbenchSSOService) IssueTicketForWorkbenchID(ctx context.Context, userID int64, workbenchID string) (*WorkbenchSSOTicket, error) {
	if s == nil || s.settingService == nil {
		return nil, infraerrors.InternalServer("WORKBENCH_SSO_NOT_CONFIGURED", "workbench sso is not configured")
	}
	workbenchID = strings.TrimSpace(workbenchID)
	if workbenchID == "" {
		return nil, infraerrors.BadRequest("WORKBENCH_ID_REQUIRED", "workbench_id is required")
	}
	tabs, err := s.settingService.GetXimoAIHomeTabs(ctx)
	if err != nil {
		return nil, err
	}
	for _, tab := range tabs {
		if tab.ID != workbenchID {
			continue
		}
		if !tab.Enabled || !tab.WorkbenchSSO {
			break
		}
		return s.IssueTicket(ctx, userID, tab.URL)
	}
	return nil, infraerrors.Forbidden("WORKBENCH_SSO_FORBIDDEN", "workbench is unavailable")
}
