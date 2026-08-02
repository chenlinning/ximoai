package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *WorkbenchSSOService) IssueTicketForWorkbenchID(ctx context.Context, userID int64, workbenchID string) (*WorkbenchSSOTicket, error) {
	audience, err := s.ResolveWorkbenchForUser(ctx, userID, workbenchID)
	if err != nil {
		return nil, err
	}
	return s.IssueTicket(ctx, userID, audience)
}

func (s *WorkbenchSSOService) ResolveWorkbenchForUser(ctx context.Context, userID int64, workbenchID string) (string, error) {
	if s == nil || s.settingService == nil {
		return "", infraerrors.InternalServer("WORKBENCH_SSO_NOT_CONFIGURED", "workbench sso is not configured")
	}
	workbenchID = strings.TrimSpace(workbenchID)
	if workbenchID == "" {
		return "", infraerrors.BadRequest("WORKBENCH_ID_REQUIRED", "workbench_id is required")
	}
	tabs, err := s.settingService.GetXimoAIHomeTabs(ctx)
	if err != nil {
		return "", err
	}
	for _, tab := range tabs {
		if tab.ID != workbenchID {
			continue
		}
		if !tab.Enabled || !tab.WorkbenchSSO {
			break
		}
		audience := normalizeWorkbenchAudience(tab.URL)
		if audience == "" {
			break
		}
		if tab.DiamondOnly {
			if err := s.requireDiamondMembership(ctx, userID); err != nil {
				return "", err
			}
		}
		return audience, nil
	}
	return "", infraerrors.Forbidden("WORKBENCH_SSO_FORBIDDEN", "workbench is unavailable")
}
