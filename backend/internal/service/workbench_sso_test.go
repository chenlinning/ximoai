//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

const workbenchTestMasterSecret = "test-workbench-master-secret-32-bytes-long"

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

type workbenchAnnouncementListerStub struct {
	items []UserAnnouncement
}

func (s workbenchAnnouncementListerStub) ListForUser(context.Context, int64, bool) ([]UserAnnouncement, error) {
	return s.items, nil
}

type workbenchTestTicketStore struct {
	mu    sync.Mutex
	now   time.Time
	items map[string]workbenchTestTicketItem
}

type workbenchTestTicketItem struct {
	value     string
	expiresAt time.Time
}

func newWorkbenchTestTicketStore() *workbenchTestTicketStore {
	return &workbenchTestTicketStore{
		now:   time.Now(),
		items: make(map[string]workbenchTestTicketItem),
	}
}

func (s *workbenchTestTicketStore) StoreTicket(_ context.Context, key string, payload []byte, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[key]; exists {
		return false, nil
	}
	s.items[key] = workbenchTestTicketItem{
		value:     string(payload),
		expiresAt: s.now.Add(ttl),
	}
	return true, nil
}

func (s *workbenchTestTicketStore) ConsumeTicket(_ context.Context, key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, exists := s.items[key]
	if !exists || !s.now.Before(item.expiresAt) {
		delete(s.items, key)
		return "", false, nil
	}
	delete(s.items, key)
	return item.value, true, nil
}

func (s *workbenchTestTicketStore) Ping(context.Context) error {
	return nil
}

func (s *workbenchTestTicketStore) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.items))
	for key := range s.items {
		keys = append(keys, key)
	}
	return keys
}

func (s *workbenchTestTicketStore) Exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.items[key]
	return exists
}

func (s *workbenchTestTicketStore) FastForward(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = s.now.Add(duration)
}

func newWorkbenchSSOTestService(t *testing.T, values map[string]string) (*WorkbenchSSOService, *workbenchTestTicketStore) {
	t.Helper()
	ticketStore := newWorkbenchTestTicketStore()
	settingSvc := NewSettingService(&settingPublicRepoStub{values: values}, &config.Config{
		Server: config.ServerConfig{
			FrontendURL: "https://ximoai.cn",
		},
		WorkbenchSSO: config.WorkbenchSSOConfig{
			TicketTTLSeconds: 60,
			InternalSecret:   workbenchTestMasterSecret,
		},
	})
	settingSvc.cfg.JWT.Secret = "test-jwt-secret-32bytes-long!!!"
	user := &User{
		ID:           123,
		Email:        "alice@example.com",
		Username:     "alice",
		AvatarURL:    "https://cdn.example.com/alice.png",
		Balance:      12.34,
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 3,
	}
	userGetter := workbenchUserGetterStub{user: user}
	controlTokens := &WorkbenchControlTokenService{
		authService: &AuthService{cfg: settingSvc.cfg},
		userGetter:  userGetter,
		grantStore:  newWorkbenchControlGrantStoreStub(),
	}
	svc := &WorkbenchSSOService{
		cfg:            settingSvc.cfg,
		settingService: settingSvc,
		ticketStore:    ticketStore,
		userGetter:     userGetter,
		membershipGetter: workbenchMembershipGetterStub{summary: &MembershipSummary{
			Level: &MembershipLevel{Code: "diamond"},
			ManagedKeys: []MembershipManagedKey{
				{
					ID:      1,
					GroupID: 8,
					Status:  ManagedKeyStatusActive,
					APIKey: &APIKey{
						Key:    "sk-openai",
						Status: StatusAPIKeyActive,
					},
				},
				{
					ID:      2,
					GroupID: 14,
					Status:  ManagedKeyStatusActive,
					APIKey: &APIKey{
						Key:    "sk-gemini",
						Status: StatusAPIKeyActive,
					},
				},
				{
					ID:      3,
					GroupID: 15,
					Status:  ManagedKeyStatusDisabled,
					APIKey: &APIKey{
						Key:    "sk-disabled",
						Status: StatusAPIKeyDisabled,
					},
				},
			},
		}},
		announcementLister: workbenchAnnouncementListerStub{items: []UserAnnouncement{
			{
				Announcement: Announcement{
					ID:        7,
					Title:     "System notice",
					Content:   "Hello Workbench",
					CreatedAt: time.Unix(1700000000, 0),
				},
			},
		}},
		controlTokens: controlTokens,
	}
	return svc, ticketStore
}

func TestWorkbenchSSOService_IssueTicketStoresOnlyHash(t *testing.T) {
	svc, mr := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"workbench","label":"Workbench","url":"http://127.0.0.1:4173/app","enabled":true,"workbench_sso":true}]`,
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
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"workbench","label":"Workbench","url":"http://127.0.0.1:4173/app","enabled":true,"workbench_sso":true}]`,
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)

	userContext, err := svc.ValidateTicket(context.Background(), ticket.Ticket, "http://127.0.0.1:4173")
	require.NoError(t, err)
	require.Equal(t, "123", userContext.UserID)
	require.Equal(t, "alice@example.com", userContext.Email)
	require.Equal(t, "alice", userContext.Username)
	require.Equal(t, "alice", userContext.DisplayName)
	require.Equal(t, "https://cdn.example.com/alice.png", userContext.AvatarURL)
	require.Equal(t, 12.34, userContext.Balance)
	require.Len(t, userContext.Announcements, 1)
	require.Equal(t, int64(7), userContext.Announcements[0].ID)
	require.Equal(t, "System notice", userContext.Announcements[0].Title)
	require.Equal(t, RoleUser, userContext.Role)
	require.Equal(t, "diamond", userContext.MembershipStatus)
	require.NotNil(t, userContext.Quota)
	require.NotNil(t, userContext.FeatureFlags)
	require.NotNil(t, userContext.Permissions)
	require.NotNil(t, userContext.Authorization)
	require.NotEmpty(t, userContext.Authorization.AccessToken)
	require.NotEmpty(t, userContext.Authorization.RefreshToken)
	require.Equal(t, WorkbenchControlAudience, userContext.Authorization.Audience)
	payload, err := json.Marshal(userContext)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "modelConfig")
	require.NotContains(t, string(payload), "sk-openai")

	_, err = svc.ValidateTicket(context.Background(), ticket.Ticket, "http://127.0.0.1:4173")
	require.Error(t, err)
}

func TestWorkbenchSSOService_ValidateTicketRejectsExpiredTicket(t *testing.T) {
	svc, mr := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "1",
		SettingKeyXimoAIHomeTabs:            `[{"id":"workbench","label":"Workbench","url":"http://127.0.0.1:4173/app","enabled":true,"workbench_sso":true}]`,
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)
	mr.FastForward(2 * time.Second)

	_, err = svc.ValidateTicket(context.Background(), ticket.Ticket, "http://127.0.0.1:4173")
	require.Error(t, err)
}

func TestWorkbenchSSOService_RejectsAudienceMismatch(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"workbench","label":"Workbench","url":"http://127.0.0.1:4173/app","enabled":true,"workbench_sso":true}]`,
	})

	ticket, err := svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.NoError(t, err)

	_, err = svc.ValidateTicket(context.Background(), ticket.Ticket, "https://evil.example")
	require.Error(t, err)
}

func TestWorkbenchSSOService_IssueTicketUsesSelectedHomeTabOrigin(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs: `[
			{"id":"workbench","label":"Workbench","url":"https://workbench.ximoai.cn/app","enabled":true,"workbench_sso":true},
			{"id":"novel","label":"Novel","url":"https://novel.ximoai.cn/workspace","enabled":true,"workbench_sso":true},
			{"id":"plain","label":"Plain","url":"https://plain.ximoai.cn/","enabled":true},
			{"id":"disabled","label":"Disabled","url":"https://disabled.ximoai.cn/","enabled":false,"workbench_sso":true}
		]`,
	})

	novelTicket, err := svc.IssueTicket(context.Background(), 123, "https://novel.ximoai.cn/workspace")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(novelTicket.EntryURL, "https://novel.ximoai.cn/sso/entry?ticket="))
	novelContext, err := svc.ValidateTicket(context.Background(), novelTicket.Ticket, "https://novel.ximoai.cn")
	require.NoError(t, err)
	require.Equal(t, "123", novelContext.UserID)

	workbenchTicket, err := svc.IssueTicket(context.Background(), 123, "https://workbench.ximoai.cn/app")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(workbenchTicket.EntryURL, "https://workbench.ximoai.cn/sso/entry?ticket="))

	_, err = svc.IssueTicket(context.Background(), 123, "https://plain.ximoai.cn/")
	require.Error(t, err)
	_, err = svc.IssueTicket(context.Background(), 123, "https://disabled.ximoai.cn/")
	require.Error(t, err)
	_, err = svc.IssueTicket(context.Background(), 123, "http://127.0.0.1:4173")
	require.Error(t, err)
}

func TestWorkbenchSSOService_RejectsDiamondOnlyTabForNonDiamondMember(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyWorkbenchTicketTTLSeconds: "60",
		SettingKeyXimoAIHomeTabs:            `[{"id":"workbench","label":"Workbench","url":"https://workbench.ximoai.cn/app","enabled":true,"workbench_sso":true,"diamond_only":true}]`,
	})
	svc.membershipGetter = workbenchMembershipGetterStub{
		summary: &MembershipSummary{Level: &MembershipLevel{Code: "platinum"}},
	}

	_, err := svc.IssueTicket(context.Background(), 123, "https://workbench.ximoai.cn")
	require.Error(t, err)
}

func TestWorkbenchSSOService_AudienceCredentialsAreIsolated(t *testing.T) {
	svc, _ := newWorkbenchSSOTestService(t, map[string]string{
		SettingKeyXimoAIHomeTabs: `[
			{"id":"workbench","label":"Workbench","url":"https://workbench.ximoai.cn","enabled":true,"workbench_sso":true},
			{"id":"novel","label":"Novel","url":"https://novel.ximoai.cn","enabled":true,"workbench_sso":true}
		]`,
	})

	workbenchSecret, err := DeriveWorkbenchAudienceSecret(workbenchTestMasterSecret, "https://workbench.ximoai.cn")
	require.NoError(t, err)
	novelSecret, err := DeriveWorkbenchAudienceSecret(workbenchTestMasterSecret, "https://novel.ximoai.cn")
	require.NoError(t, err)
	require.NotEqual(t, workbenchSecret, novelSecret)
	require.True(t, svc.AuthorizeAudience(context.Background(), "https://workbench.ximoai.cn", workbenchSecret))
	require.False(t, svc.AuthorizeAudience(context.Background(), "https://workbench.ximoai.cn", novelSecret))
	require.False(t, svc.AuthorizeAudience(context.Background(), "https://disabled.ximoai.cn", workbenchSecret))
}
