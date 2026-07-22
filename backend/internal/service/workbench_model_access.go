package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// WorkbenchModelAccessService resolves managed gateway credentials outside the SSO exchange.
type WorkbenchModelAccessService struct {
	cfg              *config.Config
	userGetter       workbenchUserGetter
	membershipGetter workbenchMembershipGetter
}

func NewWorkbenchModelAccessService(
	cfg *config.Config,
	userService *UserService,
	membershipService *MembershipService,
) *WorkbenchModelAccessService {
	return &WorkbenchModelAccessService{
		cfg:              cfg,
		userGetter:       userService,
		membershipGetter: membershipService,
	}
}

func (s *WorkbenchModelAccessService) GetUserModelConfig(ctx context.Context, userID int64) (map[string]any, error) {
	if s == nil || s.userGetter == nil || s.membershipGetter == nil {
		return nil, infraerrors.InternalServer("WORKBENCH_MODEL_ACCESS_UNAVAILABLE", "workbench model access is unavailable")
	}
	if userID <= 0 {
		return nil, infraerrors.BadRequest("WORKBENCH_MODEL_ACCESS_USER_INVALID", "user id is invalid")
	}
	user, err := s.userGetter.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.ID != userID || !user.IsActive() {
		return nil, infraerrors.Unauthorized("WORKBENCH_MODEL_ACCESS_USER_INVALID", "user is unavailable")
	}
	summary, err := s.membershipGetter.GetUserMembership(ctx, userID)
	if err != nil {
		return nil, err
	}
	return buildWorkbenchModelConfig(s.cfg, summary), nil
}

func (s *WorkbenchModelAccessService) InternalSecret() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.WorkbenchSSO.InternalSecret)
}

func buildWorkbenchModelConfig(cfg *config.Config, summary *MembershipSummary) map[string]any {
	if cfg == nil || summary == nil {
		return map[string]any{}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.Server.FrontendURL), "/")
	if baseURL == "" {
		return map[string]any{}
	}
	gateways := make([]map[string]any, 0, len(summary.ManagedKeys))
	for _, managed := range summary.ManagedKeys {
		if managed.Status != ManagedKeyStatusActive || managed.APIKey == nil || managed.APIKey.Status != StatusAPIKeyActive {
			continue
		}
		apiKey := strings.TrimSpace(managed.APIKey.Key)
		if apiKey == "" {
			continue
		}
		gateways = append(gateways, map[string]any{
			"id":      fmt.Sprintf("membership-%d", managed.ID),
			"baseUrl": baseURL,
			"apiKey":  apiKey,
			"groupId": managed.GroupID,
		})
	}
	if len(gateways) == 0 {
		return map[string]any{}
	}
	return map[string]any{"gateways": gateways}
}
