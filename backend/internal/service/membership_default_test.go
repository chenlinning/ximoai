package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type defaultMembershipRepoStub struct {
	active       *UserMembership
	defaultLevel *MembershipLevel
	upsertCalled bool
}

func (s *defaultMembershipRepoStub) ListMembershipLevels(context.Context, bool) ([]MembershipLevel, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) GetMembershipLevel(_ context.Context, id int64) (*MembershipLevel, error) {
	if s.defaultLevel != nil && s.defaultLevel.ID == id {
		return s.defaultLevel, nil
	}
	return nil, ErrMembershipLevelNotFound
}

func (s *defaultMembershipRepoStub) GetDefaultMembershipLevel(context.Context) (*MembershipLevel, error) {
	if s.defaultLevel == nil {
		return nil, ErrMembershipDefaultNotFound
	}
	return s.defaultLevel, nil
}

func (s *defaultMembershipRepoStub) CreateMembershipLevel(context.Context, MembershipLevelInput) (*MembershipLevel, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) UpdateMembershipLevel(context.Context, int64, MembershipLevelInput) (*MembershipLevel, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) DisableMembershipLevel(context.Context, int64) error {
	return nil
}

func (s *defaultMembershipRepoStub) ListMembershipLevelsByGroup(context.Context, int64) ([]MembershipLevel, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) GetActiveUserMembership(context.Context, int64) (*UserMembership, error) {
	if s.active == nil {
		return nil, ErrMembershipLevelNotFound
	}
	return s.active, nil
}

func (s *defaultMembershipRepoStub) UpsertActiveUserMembership(_ context.Context, userID, levelID int64, startsAt time.Time, expiresAt *time.Time, source string) (*UserMembership, error) {
	s.upsertCalled = true
	s.active = &UserMembership{
		ID:                100,
		UserID:            userID,
		MembershipLevelID: levelID,
		StartsAt:          startsAt,
		ExpiresAt:         expiresAt,
		Status:            MembershipStatusActive,
		Source:            source,
	}
	return s.active, nil
}

func (s *defaultMembershipRepoStub) MarkUserMembershipExpired(context.Context, int64) error {
	return nil
}

func (s *defaultMembershipRepoStub) ListUserMembershipsByLevel(context.Context, int64) ([]UserMembership, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) ListExpiredActiveUserMemberships(context.Context, time.Time, int) ([]UserMembership, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) ListActiveUserMemberships(context.Context, int) ([]UserMembership, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) ListManagedKeysByUser(context.Context, int64) ([]MembershipManagedKey, error) {
	return nil, nil
}

func (s *defaultMembershipRepoStub) GetManagedKeyByUserGroup(context.Context, int64, int64) (*MembershipManagedKey, error) {
	return nil, ErrAPIKeyNotFound
}

func (s *defaultMembershipRepoStub) GetManagedKeyByAPIKeyID(context.Context, int64) (*MembershipManagedKey, error) {
	return nil, ErrAPIKeyNotFound
}

func (s *defaultMembershipRepoStub) UpsertManagedKey(context.Context, MembershipManagedKey) error {
	return nil
}

func (s *defaultMembershipRepoStub) SetManagedKeyStatus(context.Context, int64, int64, string, string, int64) error {
	return nil
}

func TestAssignDefaultMembershipDoesNotOverrideExistingActiveMembership(t *testing.T) {
	repo := &defaultMembershipRepoStub{
		active: &UserMembership{
			ID:                1,
			UserID:            10,
			MembershipLevelID: 2,
			Status:            MembershipStatusActive,
		},
		defaultLevel: &MembershipLevel{ID: 1, Name: "Default", Enabled: true, IsDefault: true},
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	require.NoError(t, svc.AssignDefaultMembership(context.Background(), 10))

	require.False(t, repo.upsertCalled)
	require.EqualValues(t, 2, repo.active.MembershipLevelID)
}

func TestGetUserMembershipCreatesDefaultWhenMissing(t *testing.T) {
	repo := &defaultMembershipRepoStub{
		defaultLevel: &MembershipLevel{ID: 1, Name: "Default", Enabled: true, IsDefault: true},
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	summary, err := svc.GetUserMembership(context.Background(), 10)

	require.NoError(t, err)
	require.True(t, repo.upsertCalled)
	require.NotNil(t, summary)
	require.EqualValues(t, 1, summary.Level.ID)
	require.NotNil(t, summary.Groups)
	require.NotNil(t, summary.ManagedKeys)
	require.Empty(t, summary.Groups)
	require.Empty(t, summary.ManagedKeys)
}
