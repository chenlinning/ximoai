//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchModelAccessServiceReturnsManagedGatewayCredentialsOutsideSSO(t *testing.T) {
	service := &WorkbenchModelAccessService{
		cfg:        &config.Config{Server: config.ServerConfig{FrontendURL: "https://ximoai.cn"}},
		userGetter: workbenchUserGetterStub{user: &User{ID: 123, Status: StatusActive}},
		membershipGetter: workbenchMembershipGetterStub{summary: &MembershipSummary{
			ManagedKeys: []MembershipManagedKey{
				{
					ID:      1,
					GroupID: 8,
					Status:  ManagedKeyStatusActive,
					APIKey:  &APIKey{Key: "sk-openai", Status: StatusAPIKeyActive},
				},
				{
					ID:      2,
					GroupID: 14,
					Status:  ManagedKeyStatusDisabled,
					APIKey:  &APIKey{Key: "sk-disabled", Status: StatusAPIKeyDisabled},
				},
			},
		}},
	}

	modelConfig, err := service.GetUserModelConfig(context.Background(), 123)

	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"gateways": []map[string]any{
			{
				"id":      "membership-1",
				"baseUrl": "https://ximoai.cn",
				"apiKey":  "sk-openai",
				"groupId": int64(8),
			},
		},
	}, modelConfig)
}

func TestWorkbenchModelAccessServiceRejectsDisabledUser(t *testing.T) {
	service := &WorkbenchModelAccessService{
		cfg:              &config.Config{Server: config.ServerConfig{FrontendURL: "https://ximoai.cn"}},
		userGetter:       workbenchUserGetterStub{user: &User{ID: 123, Status: StatusDisabled}},
		membershipGetter: workbenchMembershipGetterStub{summary: &MembershipSummary{}},
	}

	_, err := service.GetUserModelConfig(context.Background(), 123)

	require.Error(t, err)
}
