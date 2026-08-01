//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchSSOService_IssueTicketForWorkbenchIDUsesConfiguredTabOnly(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs: `[
			{"id":"image","label":"Image","url":"https://image.ximoai.cn/app","enabled":true,"workbench_sso":true},
			{"id":"plain","label":"Plain","url":"https://plain.ximoai.cn","enabled":true,"workbench_sso":false},
			{"id":"disabled","label":"Disabled","url":"https://disabled.ximoai.cn","enabled":false,"workbench_sso":true}
		]`,
	})

	ticket, err := svc.IssueTicketForWorkbenchID(context.Background(), 123, "image")
	require.NoError(t, err)
	require.Contains(t, ticket.EntryURL, "https://image.ximoai.cn/sso/entry")

	for _, workbenchID := range []string{"plain", "disabled", "missing", "https://image.ximoai.cn"} {
		_, err := svc.IssueTicketForWorkbenchID(context.Background(), 123, workbenchID)
		require.Equal(t, "WORKBENCH_SSO_FORBIDDEN", infraerrors.Reason(err), "workbench_id=%s", workbenchID)
	}
}
