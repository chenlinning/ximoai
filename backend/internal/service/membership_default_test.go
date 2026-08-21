package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type defaultMembershipRepoStub struct {
	active              *UserMembership
	defaultLevel        *MembershipLevel
	levels              []MembershipLevel
	managedKey          *MembershipManagedKey
	createInput         *MembershipLevelInput
	updateInput         *MembershipLevelInput
	upsertCalled        bool
	managedUpsertCalled bool
	managedDeleteCalled bool
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

func (s *defaultMembershipRepoStub) ListManagedKeysByGroup(context.Context, int64) ([]MembershipManagedKey, error) {
	if s.managedKey != nil {
		return []MembershipManagedKey{*s.managedKey}, nil
	}
	return nil, nil
}

func (s *defaultMembershipRepoStub) GetManagedKeyByUserGroup(_ context.Context, userID, groupID int64) (*MembershipManagedKey, error) {
	if s.managedKey != nil && s.managedKey.UserID == userID && s.managedKey.GroupID == groupID {
		return s.managedKey, nil
	}
	return nil, ErrAPIKeyNotFound
}

func (s *defaultMembershipRepoStub) GetManagedKeyByAPIKeyID(context.Context, int64) (*MembershipManagedKey, error) {
	return nil, ErrAPIKeyNotFound
}

func (s *defaultMembershipRepoStub) UpsertManagedKey(_ context.Context, key MembershipManagedKey) error {
	s.managedUpsertCalled = true
	s.managedKey = &key
	return nil
}

func (s *defaultMembershipRepoStub) SetManagedKeyStatus(context.Context, int64, int64, string, string, int64) error {
	return nil
}

func (s *defaultMembershipRepoStub) DeleteManagedKey(context.Context, int64, int64) error {
	s.managedDeleteCalled = true
	s.managedKey = nil
	return nil
}

type membershipAPIKeyRepoStub struct {
	APIKeyRepository
	key        *APIKey
	createdKey *APIKey
	deletedIDs []int64
}

func (s *membershipAPIKeyRepoStub) Create(_ context.Context, key *APIKey) error {
	key.ID = 200
	s.key = key
	s.createdKey = key
	return nil
}

func (s *membershipAPIKeyRepoStub) GetByID(context.Context, int64) (*APIKey, error) {
	if s.key == nil {
		return nil, ErrAPIKeyNotFound
	}
	return s.key, nil
}

func (s *membershipAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	if s.key == nil {
		return "", 0, ErrAPIKeyNotFound
	}
	return s.key.Key, s.key.UserID, nil
}

func (s *membershipAPIKeyRepoStub) Update(_ context.Context, key *APIKey, _ APIKeyUpdateFields) error {
	s.key = key
	return nil
}

func (s *membershipAPIKeyRepoStub) DeleteWithAudit(_ context.Context, id int64) error {
	s.deletedIDs = append(s.deletedIDs, id)
	s.key = nil
	return nil
}

type membershipUserRepoStub struct {
	UserRepository
	user *User
}

func (s *membershipUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type membershipGroupRepoStub struct {
	GroupRepository
	group *Group
}

func (s *membershipGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	return s.group, nil
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
		Code:         MembershipLevelCodeSilver,
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
	require.Equal(t, "白银会员", repo.createInput.Name)
	require.Equal(t, MembershipLevelCodeSilver, repo.createInput.Code)
	require.Equal(t, defaultMembershipLevelColor, repo.createInput.Color)
	require.True(t, repo.createInput.Enabled)
	require.True(t, repo.createInput.IsDefault)
	require.Equal(t, 10, repo.createInput.SortOrder)
	require.Equal(t, "系统固定白银会员等级", repo.createInput.Description)
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
	require.True(t, repo.updateInput.IsDefault)
	require.Equal(t, 10, repo.updateInput.SortOrder)
	require.Equal(t, "系统固定白银会员等级", repo.updateInput.Description)
	require.Equal(t, []int64{10, 20}, repo.updateInput.GroupIDs)
}

func TestFixedMembershipLevelsIncludePlatinumTier(t *testing.T) {
	defs := FixedMembershipLevelDefinitions()

	require.Len(t, defs, 4)
	require.Equal(t, MembershipLevelCodeSilver, defs[0].Code)
	require.Equal(t, "白银会员", defs[0].Name)
	require.Equal(t, "#94a3b8", defs[0].Color)
	require.True(t, defs[0].IsDefault)
	require.Equal(t, MembershipLevelCodePlatinum, defs[2].Code)
	require.Equal(t, "铂金会员", defs[2].Name)
	require.Equal(t, "#b89a56", defs[2].Color)
	require.Equal(t, 30, defs[2].SortOrder)
	require.Equal(t, MembershipLevelCodeDiamond, defs[3].Code)
	require.Equal(t, 40, defs[3].SortOrder)
}

func TestManagedAPIKeyNameFollowsGroupName(t *testing.T) {
	require.Equal(t, "Membership Key - Silver Group", managedAPIKeyName("Silver Group"))
	require.Len(t, []rune(managedAPIKeyName(strings.Repeat("分", 100))), 100)
}

func TestEnsureManagedKeyRenamesExistingKeyAfterGroupRename(t *testing.T) {
	group := &Group{ID: 20, Name: "Silver Group", Status: StatusActive}
	apiKey := &APIKey{ID: 100, UserID: 10, Key: "sk-existing", Name: "Membership Key - Old Group", Status: StatusAPIKeyActive, GroupID: &group.ID}
	repo := &defaultMembershipRepoStub{managedKey: &MembershipManagedKey{
		UserID: 10, GroupID: group.ID, APIKeyID: apiKey.ID, MembershipLevelID: 1,
		Status: ManagedKeyStatusActive, APIKey: apiKey, Group: group,
	}}
	apiKeyRepo := &membershipAPIKeyRepoStub{key: apiKey}
	userRepo := &membershipUserRepoStub{user: &User{ID: 10, Status: StatusActive}}
	groupRepo := &membershipGroupRepoStub{group: group}
	apiKeyService := NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, nil, nil, nil, &config.Config{})
	svc := NewMembershipService(repo, userRepo, groupRepo, nil, apiKeyService)

	err := svc.EnsureManagedKey(context.Background(), 10, group.ID, 1)

	require.NoError(t, err)
	require.Equal(t, "Membership Key - Silver Group", apiKeyRepo.key.Name)
	require.False(t, repo.managedDeleteCalled)
}

func TestEnsureManagedKeyReplacesUnavailableManagedKey(t *testing.T) {
	group := &Group{ID: 20, Name: "Silver Group", Status: StatusActive}
	oldKey := &APIKey{ID: 100, UserID: 10, Key: "sk-disabled", Name: "Membership Key - Silver Group", Status: StatusAPIKeyDisabled, GroupID: &group.ID}
	repo := &defaultMembershipRepoStub{managedKey: &MembershipManagedKey{
		UserID: 10, GroupID: group.ID, APIKeyID: oldKey.ID, MembershipLevelID: 1,
		Status: ManagedKeyStatusDisabled, APIKey: oldKey, Group: group,
	}}
	apiKeyRepo := &membershipAPIKeyRepoStub{key: oldKey}
	userRepo := &membershipUserRepoStub{user: &User{ID: 10, Status: StatusActive}}
	groupRepo := &membershipGroupRepoStub{group: group}
	apiKeyService := NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, nil, nil, nil, &config.Config{})
	svc := NewMembershipService(repo, userRepo, groupRepo, nil, apiKeyService)

	err := svc.EnsureManagedKey(context.Background(), 10, group.ID, 1)

	require.NoError(t, err)
	require.Equal(t, []int64{oldKey.ID}, apiKeyRepo.deletedIDs)
	require.True(t, repo.managedDeleteCalled)
	require.True(t, repo.managedUpsertCalled)
	require.NotNil(t, apiKeyRepo.createdKey)
	require.Equal(t, StatusAPIKeyActive, apiKeyRepo.createdKey.Status)
	require.Equal(t, "Membership Key - Silver Group", apiKeyRepo.createdKey.Name)
}
