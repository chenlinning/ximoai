//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestDesktopSSOBrokerCredentialIsBoundToParentSessionUserAndWorkbench(t *testing.T) {
	desktop, store, users := newDesktopSessionTestService(t)
	workbench, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"image","label":"Image","url":"https://image.ximoai.cn/app","enabled":true,"workbench_sso":true}]`,
	})

	now := time.Now().Truncate(time.Second)
	session := desktopSessionRecord{
		SessionID:         "desktop-session",
		UserID:            users.user.ID,
		TokenVersion:      resolvedTokenVersion(users.user),
		JKT:               "device-thumbprint",
		CurrentRefreshKey: "desktop:refresh:key",
		ExpiresAt:         now.Add(time.Hour).Unix(),
	}
	payload, err := json.Marshal(session)
	require.NoError(t, err)
	store.sessions[desktopSessionKey(session.SessionID)] = string(payload)

	broker := NewDesktopSSOBrokerService(desktop.authService, desktop, workbench)
	broker.now = func() time.Time { return now }
	credential, err := broker.Issue(context.Background(), &DesktopIdentity{
		UserID:       session.UserID,
		SessionID:    session.SessionID,
		TokenVersion: session.TokenVersion,
		JKT:          session.JKT,
	}, "image")
	require.NoError(t, err)
	require.NotEmpty(t, credential.Credential)
	require.Equal(t, "Bearer", credential.TokenType)
	require.Equal(t, int(desktopSSOBrokerCredentialTTL.Seconds()), credential.ExpiresIn)
	require.Equal(t, "image", credential.WorkbenchID)
	require.Equal(t, "https://image.ximoai.cn", credential.Audience)
	_, err = desktop.authService.ValidateToken(credential.Credential)
	require.Error(t, err)

	identity, err := broker.Authenticate(context.Background(), credential.Credential)
	require.NoError(t, err)
	require.Equal(t, session.UserID, identity.UserID)
	require.Equal(t, session.SessionID, identity.DesktopSessionID)
	require.Equal(t, "image", identity.WorkbenchID)
	require.Equal(t, "https://image.ximoai.cn", identity.Audience)

	_, _ = store.RevokeSession(context.Background(), desktopSessionKey(session.SessionID))
	_, err = broker.Authenticate(context.Background(), credential.Credential)
	require.Equal(t, "DESKTOP_SESSION_INVALID", infraerrors.Reason(err))
}

func TestDesktopSSOBrokerCredentialRechecksWorkbenchPermission(t *testing.T) {
	desktop, store, users := newDesktopSessionTestService(t)
	workbench, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"image","label":"Image","url":"https://image.ximoai.cn","enabled":true,"workbench_sso":true,"diamond_only":true}]`,
	})
	now := time.Now().Truncate(time.Second)
	session := desktopSessionRecord{
		SessionID: "desktop-session", UserID: users.user.ID, TokenVersion: resolvedTokenVersion(users.user),
		JKT: "device-thumbprint", CurrentRefreshKey: "desktop:refresh:key", ExpiresAt: now.Add(time.Hour).Unix(),
	}
	payload, err := json.Marshal(session)
	require.NoError(t, err)
	store.sessions[desktopSessionKey(session.SessionID)] = string(payload)
	broker := NewDesktopSSOBrokerService(desktop.authService, desktop, workbench)
	broker.now = func() time.Time { return now }

	credential, err := broker.Issue(context.Background(), &DesktopIdentity{
		UserID: session.UserID, SessionID: session.SessionID, TokenVersion: session.TokenVersion, JKT: session.JKT,
	}, "image")
	require.NoError(t, err)

	workbench.membershipGetter = workbenchMembershipGetterStub{
		summary: &MembershipSummary{Level: &MembershipLevel{Code: "platinum"}},
	}
	_, err = broker.Authenticate(context.Background(), credential.Credential)
	require.Equal(t, "WORKBENCH_SSO_DIAMOND_REQUIRED", infraerrors.Reason(err))
}

func TestBrokerBoundTicketValidationDoesNotConsumeAnotherUsersTicket(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"image","label":"Image","url":"https://image.ximoai.cn","enabled":true,"workbench_sso":true}]`,
	})
	ticket, err := svc.IssueTicket(context.Background(), 123, "https://image.ximoai.cn")
	require.NoError(t, err)

	_, err = svc.ValidateTicketForUser(context.Background(), ticket.Ticket, "https://image.ximoai.cn", 456)
	require.Equal(t, "WORKBENCH_SSO_TICKET_INVALID", infraerrors.Reason(err))

	validated, err := svc.ValidateTicketForUser(context.Background(), ticket.Ticket, "https://image.ximoai.cn", 123)
	require.NoError(t, err)
	require.Equal(t, "123", validated.UserID)
}

func TestBrokerBoundControlGrantCannotBeConsumedOrRevokedByAnotherUser(t *testing.T) {
	control, _, _ := newWorkbenchControlTokenTestService()
	grant, err := control.Issue(context.Background(), 123, "https://image.ximoai.cn")
	require.NoError(t, err)

	_, err = control.RefreshForUser(context.Background(), grant.RefreshToken, "https://image.ximoai.cn", 456)
	require.Equal(t, "WORKBENCH_CONTROL_REFRESH_INVALID", infraerrors.Reason(err))
	rotated, err := control.RefreshForUser(context.Background(), grant.RefreshToken, "https://image.ximoai.cn", 123)
	require.NoError(t, err)

	err = control.RevokeForUser(context.Background(), rotated.RefreshToken, "https://image.ximoai.cn", 456)
	require.Equal(t, "WORKBENCH_CONTROL_REFRESH_INVALID", infraerrors.Reason(err))
	require.NoError(t, control.RevokeForUser(context.Background(), rotated.RefreshToken, "https://image.ximoai.cn", 123))
}
