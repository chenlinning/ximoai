//go:build unit

package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
		ticketStore:    ticketStore,
		userGetter: workbenchUserGetterStub{user: &User{
			ID:        123,
			Email:     "alice@example.com",
			Username:  "alice",
			AvatarURL: "https://cdn.example.com/alice.png",
			Balance:   12.34,
			Role:      RoleUser,
		}},
		membershipGetter: workbenchMembershipGetterStub{summary: &MembershipSummary{
			Level: &MembershipLevel{Code: "diamond"},
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
	}
	return svc, ticketStore
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
