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
	levels       []MembershipLevel
	createInput  *MembershipLevelInput
	updateInput  *MembershipLevelInput
	upsertCalled bool
}

type expiringMembershipRepoStub struct {
	defaultMembershipRepoStub
	batches      [][]UserMembership
	listCalls    int
	expiredIDs   []int64
	activeByUser map[int64]*UserMembership
	idToUser     map[int64]int64
}

func (s *defaultMembershipRepoStub) ListMembershipLevels(context.Context, bool) ([]MembershipLevel, error) {
	return s.levels, nil
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

func (s *defaultMembershipRepoStub) CreateMembershipLevel(_ context.Context, input MembershipLevelInput) (*MembershipLevel, error) {
	s.createInput = &input
	return &MembershipLevel{
		ID:           1,
		Name:         input.Name,
		Code:         input.Code,
		Color:        input.Color,
		DiscountRate: input.DiscountRate,
		Enabled:      input.Enabled,
		IsDefault:    input.IsDefault,
		SortOrder:    input.SortOrder,
		Description:  input.Description,
	}, nil
}

func (s *defaultMembershipRepoStub) UpdateMembershipLevel(_ context.Context, id int64, input MembershipLevelInput) (*MembershipLevel, error) {
	s.updateInput = &input
	return &MembershipLevel{
		ID:           id,
		Name:         input.Name,
		Code:         input.Code,
		Color:        input.Color,
		DiscountRate: input.DiscountRate,
		Enabled:      input.Enabled,
		IsDefault:    input.IsDefault,
		SortOrder:    input.SortOrder,
		Description:  input.Description,
	}, nil
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

func (s *defaultMembershipRepoStub) ListActiveUserMembershipsAfterID(context.Context, int64, int) ([]UserMembership, error) {
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

func (s *expiringMembershipRepoStub) GetActiveUserMembership(_ context.Context, userID int64) (*UserMembership, error) {
	if s.activeByUser != nil {
		if membership := s.activeByUser[userID]; membership != nil {
			return membership, nil
		}
	}
	return nil, ErrMembershipLevelNotFound
}

func (s *expiringMembershipRepoStub) UpsertActiveUserMembership(_ context.Context, userID, levelID int64, startsAt time.Time, expiresAt *time.Time, source string) (*UserMembership, error) {
	if s.activeByUser == nil {
		s.activeByUser = make(map[int64]*UserMembership)
	}
	membership := &UserMembership{
		ID:                1000 + userID,
		UserID:            userID,
		MembershipLevelID: levelID,
		StartsAt:          startsAt,
		ExpiresAt:         expiresAt,
		Status:            MembershipStatusActive,
		Source:            source,
	}
	s.activeByUser[userID] = membership
	return membership, nil
}

func (s *expiringMembershipRepoStub) MarkUserMembershipExpired(_ context.Context, membershipID int64) error {
	s.expiredIDs = append(s.expiredIDs, membershipID)
	if s.activeByUser != nil && s.idToUser != nil {
		delete(s.activeByUser, s.idToUser[membershipID])
	}
	return nil
}

func (s *expiringMembershipRepoStub) ListExpiredActiveUserMemberships(context.Context, time.Time, int) ([]UserMembership, error) {
	s.listCalls++
	if len(s.batches) == 0 {
		return nil, nil
	}
	batch := s.batches[0]
	s.batches = s.batches[1:]
	if s.idToUser == nil {
		s.idToUser = make(map[int64]int64)
	}
	for _, membership := range batch {
		s.idToUser[membership.ID] = membership.UserID
	}
	return batch, nil
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
		levels:       []MembershipLevel{{ID: 1, Name: "Default", Enabled: true, IsDefault: true}},
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	summary, err := svc.GetUserMembership(context.Background(), 10)

	require.NoError(t, err)
	require.True(t, repo.upsertCalled)
	require.NotNil(t, summary)
	require.EqualValues(t, 1, summary.Level.ID)
	require.NotNil(t, summary.Groups)
	require.NotNil(t, summary.ManagedKeys)
	require.NotNil(t, summary.Levels)
	require.Len(t, summary.Levels, 1)
	require.Empty(t, summary.Groups)
	require.Empty(t, summary.ManagedKeys)
}

func TestExpireMembershipsDrainsExpiredBatches(t *testing.T) {
	repo := &expiringMembershipRepoStub{
		defaultMembershipRepoStub: defaultMembershipRepoStub{
			defaultLevel: &MembershipLevel{ID: 1, Name: "Default", Enabled: true, IsDefault: true},
		},
		batches: [][]UserMembership{
			{
				{ID: 11, UserID: 101, MembershipLevelID: 2, Status: MembershipStatusActive},
				{ID: 12, UserID: 102, MembershipLevelID: 2, Status: MembershipStatusActive},
			},
			{
				{ID: 13, UserID: 103, MembershipLevelID: 2, Status: MembershipStatusActive},
			},
			nil,
		},
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	require.NoError(t, svc.ExpireMemberships(context.Background()))

	require.Equal(t, 3, repo.listCalls)
	require.Equal(t, []int64{11, 12, 13}, repo.expiredIDs)
}

func TestCreateMembershipLevelAppliesFixedMetadata(t *testing.T) {
	repo := &defaultMembershipRepoStub{}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	level, err := svc.CreateLevel(context.Background(), MembershipLevelInput{
		Name:         "Custom",
		Code:         MembershipLevelCodeBronze,
		Color:        "green",
		DiscountRate: 0.8,
		Enabled:      false,
		IsDefault:    false,
		SortOrder:    99,
		Description:  "custom",
	})

	require.NoError(t, err)
	require.NotNil(t, level)
	require.NotNil(t, repo.createInput)
	require.Equal(t, "黄铜会员", repo.createInput.Name)
	require.Equal(t, MembershipLevelCodeBronze, repo.createInput.Code)
	require.Equal(t, defaultMembershipLevelColor, repo.createInput.Color)
	require.True(t, repo.createInput.Enabled)
	require.True(t, repo.createInput.IsDefault)
	require.Equal(t, 10, repo.createInput.SortOrder)
	require.Equal(t, "系统固定黄铜会员等级", repo.createInput.Description)
	require.Equal(t, 0.8, repo.createInput.DiscountRate)
}

func TestCreateMembershipLevelRejectsNonFixedCode(t *testing.T) {
	repo := &defaultMembershipRepoStub{}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	_, err := svc.CreateLevel(context.Background(), MembershipLevelInput{
		Name:         "VIP",
		Code:         "vip",
		DiscountRate: 1,
		Enabled:      true,
	})

	require.ErrorIs(t, err, ErrMembershipFixedLevelsOnly)
	require.Nil(t, repo.createInput)
}

func TestUpdateMembershipLevelKeepsFixedMetadata(t *testing.T) {
	repo := &defaultMembershipRepoStub{
		defaultLevel: &MembershipLevel{
			ID:           2,
			Name:         "Old",
			Code:         MembershipLevelCodeSilver,
			Color:        "#000000",
			DiscountRate: 1,
			Enabled:      true,
		},
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	level, err := svc.UpdateLevel(context.Background(), 2, MembershipLevelInput{
		Name:         "Hacked",
		Code:         "hacked",
		Color:        "#ffffff",
		DiscountRate: 0.75,
		Enabled:      false,
		IsDefault:    true,
		SortOrder:    99,
		Description:  "hacked",
		GroupIDs:     []int64{10, 20},
	})

	require.NoError(t, err)
	require.NotNil(t, level)
	require.NotNil(t, repo.updateInput)
	require.Equal(t, "白银会员", repo.updateInput.Name)
	require.Equal(t, MembershipLevelCodeSilver, repo.updateInput.Code)
	require.Equal(t, "#94a3b8", repo.updateInput.Color)
	require.Equal(t, 0.75, repo.updateInput.DiscountRate)
	require.True(t, repo.updateInput.Enabled)
	require.False(t, repo.updateInput.IsDefault)
	require.Equal(t, 20, repo.updateInput.SortOrder)
	require.Equal(t, "系统固定白银会员等级", repo.updateInput.Description)
	require.Equal(t, []int64{10, 20}, repo.updateInput.GroupIDs)
}

func TestFixedMembershipLevelsIncludePlatinumTier(t *testing.T) {
	defs := FixedMembershipLevelDefinitions()

	require.Len(t, defs, 5)
	require.Equal(t, MembershipLevelCodeBronze, defs[0].Code)
	require.Equal(t, "黄铜会员", defs[0].Name)
	require.Equal(t, "#a15a2b", defs[0].Color)
	require.Equal(t, MembershipLevelCodePlatinum, defs[3].Code)
	require.Equal(t, "铂金会员", defs[3].Name)
	require.Equal(t, "#b89a56", defs[3].Color)
	require.Equal(t, 40, defs[3].SortOrder)
	require.Equal(t, MembershipLevelCodeDiamond, defs[4].Code)
	require.Equal(t, 50, defs[4].SortOrder)
}

func TestShouldRestoreManagedAPIKeyOnlyForMembershipDisabledReasons(t *testing.T) {
	tests := []struct {
		name    string
		key     *MembershipManagedKey
		restore bool
	}{
		{
			name:    "active key does not need status restore",
			key:     &MembershipManagedKey{Status: ManagedKeyStatusActive, APIKey: &APIKey{Status: StatusAPIKeyActive}},
			restore: false,
		},
		{
			name:    "membership expired disabled key can restore",
			key:     &MembershipManagedKey{Status: ManagedKeyStatusDisabled, DisabledReason: ManagedKeyDisabledMembershipExpired, APIKey: &APIKey{Status: StatusAPIKeyDisabled}},
			restore: true,
		},
		{
			name:    "membership group removed disabled key can restore",
			key:     &MembershipManagedKey{Status: ManagedKeyStatusDisabled, DisabledReason: ManagedKeyDisabledGroupRemoved, APIKey: &APIKey{Status: StatusAPIKeyDisabled}},
			restore: true,
		},
		{
			name:    "manual disabled key stays disabled",
			key:     &MembershipManagedKey{Status: ManagedKeyStatusActive, DisabledReason: "", APIKey: &APIKey{Status: StatusAPIKeyDisabled}},
			restore: false,
		},
		{
			name:    "quota exhausted key stays exhausted",
			key:     &MembershipManagedKey{Status: ManagedKeyStatusActive, APIKey: &APIKey{Status: StatusAPIKeyQuotaExhausted}},
			restore: false,
		},
		{
			name:    "expired api key stays expired",
			key:     &MembershipManagedKey{Status: ManagedKeyStatusActive, APIKey: &APIKey{Status: StatusAPIKeyExpired}},
			restore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.restore, shouldRestoreManagedAPIKey(tt.key))
		})
	}
}
