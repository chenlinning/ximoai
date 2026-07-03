//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type workbenchUserGetterStub struct {
	user *User
}

func (s workbenchUserGetterStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type workbenchMembershipGetterStub struct {
	summary *MembershipSummary
}

func (s workbenchMembershipGetterStub) GetUserMembership(context.Context, int64) (*MembershipSummary, error) {
	return s.summary, nil
}

func newWorkbenchSSOTestService(t *testing.T, values map[string]string) (*WorkbenchSSOService, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	settingSvc := NewSettingService(&settingPublicRepoStub{values: values}, &config.Config{
		WorkbenchSSO: config.WorkbenchSSOConfig{
			Enabled:          true,
			BaseURL:          "http://127.0.0.1:4173",
			TicketTTLSeconds: 60,
			InternalSecret:   "secret",
		},
	})
	svc := &WorkbenchSSOService{
		cfg:            settingSvc.cfg,
		settingService: settingSvc,
		redisClient:    rdb,
		userGetter:     workbenchUserGetterStub{user: &User{ID: 123, Role: RoleUser}},
		membershipGetter: workbenchMembershipGetterStub{summary: &MembershipSummary{
			Level: &MembershipLevel{Code: "diamond"},
		}},
	}
	return svc, mr
}

func TestWorkbenchSSOService_IssueTicketStoresOnlyHash(t *testing.T) {
	svc, mr := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchSSOEnabled:       "true",
		SettingKeyWorkbenchBaseURL:          "http://127.0.0.1:4173",
		SettingKeyWorkbenchTicketTTLSeconds: "60",
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)
	require.NotEmpty(t, ticket.Ticket)
	require.Equal(t, 60, ticket.ExpiresIn)
	require.Contains(t, ticket.EntryURL, "/sso/entry?ticket=")

	keys := mr.Keys()
	require.Len(t, keys, 1)
	require.True(t, strings.HasPrefix(keys[0], workbenchSSOTicketKeyPrefix))
	require.NotContains(t, keys[0], ticket.Ticket)
	require.False(t, mr.Exists(workbenchSSOTicketKeyPrefix+ticket.Ticket))
}

func TestWorkbenchSSOService_ValidateTicketConsumesOnceAndReturnsContext(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchSSOEnabled:       "true",
		SettingKeyWorkbenchBaseURL:          "http://127.0.0.1:4173",
		SettingKeyWorkbenchTicketTTLSeconds: "60",
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)

	userContext, err := svc.ValidateTicket(context.Background(), ticket.Ticket, "http://127.0.0.1:4173")
	require.NoError(t, err)
	require.Equal(t, "123", userContext.UserID)
	require.Equal(t, RoleUser, userContext.Role)
	require.Equal(t, "diamond", userContext.MembershipStatus)
	require.NotNil(t, userContext.Quota)
	require.NotNil(t, userContext.ModelConfig)
	require.NotNil(t, userContext.FeatureFlags)
	require.NotNil(t, userContext.Permissions)

	_, err = svc.ValidateTicket(context.Background(), ticket.Ticket, "http://127.0.0.1:4173")
	require.Error(t, err)
}

func TestWorkbenchSSOService_ValidateTicketRejectsExpiredTicket(t *testing.T) {
	svc, mr := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchSSOEnabled:       "true",
		SettingKeyWorkbenchBaseURL:          "http://127.0.0.1:4173",
		SettingKeyWorkbenchTicketTTLSeconds: "1",
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)
	mr.FastForward(2 * time.Second)

	_, err = svc.ValidateTicket(context.Background(), ticket.Ticket, "http://127.0.0.1:4173")
	require.Error(t, err)
}

func TestWorkbenchSSOService_RejectsAudienceMismatch(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchSSOEnabled:       "true",
		SettingKeyWorkbenchBaseURL:          "http://127.0.0.1:4173",
		SettingKeyWorkbenchTicketTTLSeconds: "60",
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)

	_, err = svc.ValidateTicket(context.Background(), ticket.Ticket, "https://evil.example")
	require.Error(t, err)
}
