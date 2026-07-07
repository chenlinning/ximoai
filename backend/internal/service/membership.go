package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	MembershipStatusActive   = "active"
	MembershipStatusExpired  = "expired"
	MembershipStatusDisabled = "disabled"

	MembershipSourceSystem   = "system"
	MembershipSourceAdmin    = "admin"
	MembershipSourcePurchase = "purchase"

	ManagedKeyStatusActive   = "active"
	ManagedKeyStatusDisabled = "disabled"

	ManagedKeyDisabledMembershipExpired = "membership_expired"
	ManagedKeyDisabledGroupRemoved      = "membership_group_removed"
	ManagedKeyDisabledLevelDisabled     = "membership_level_disabled"
	ManagedKeyDisabledRepairDisabled    = "repair_disabled"

	defaultMembershipLevelColor = "#a15a2b"
)

var (
	ErrMembershipLevelNotFound      = infraerrors.NotFound("MEMBERSHIP_LEVEL_NOT_FOUND", "membership level not found")
	ErrMembershipDefaultNotFound    = infraerrors.NotFound("MEMBERSHIP_DEFAULT_NOT_FOUND", "default membership level not found")
	ErrMembershipInvalidDiscount    = infraerrors.BadRequest("MEMBERSHIP_INVALID_DISCOUNT", "discount_rate must be >= 0")
	ErrMembershipInvalidColor       = infraerrors.BadRequest("MEMBERSHIP_INVALID_COLOR", "color must be a valid hex color")
	ErrMembershipFixedLevelsOnly    = infraerrors.BadRequest("MEMBERSHIP_FIXED_LEVELS_ONLY", "membership levels are fixed")
	ErrMembershipManagedKeyDeletion = infraerrors.Forbidden("MEMBERSHIP_MANAGED_KEY_DELETE_FORBIDDEN", "membership managed api key cannot be deleted")
	ErrMembershipManagedKeyEnable   = infraerrors.Forbidden("MEMBERSHIP_MANAGED_KEY_ENABLE_FORBIDDEN", "membership managed api key cannot be enabled by user")
)

var membershipLevelColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type MembershipLevel struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Color        string    `json:"color"`
	DiscountRate float64   `json:"discount_rate"`
	Enabled      bool      `json:"enabled"`
	IsDefault    bool      `json:"is_default"`
	SortOrder    int       `json:"sort_order"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Groups       []Group   `json:"groups,omitempty"`
}

type UserMembership struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	MembershipLevelID int64      `json:"membership_level_id"`
	StartsAt          time.Time  `json:"starts_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	Status            string     `json:"status"`
	Source            string     `json:"source"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Level             *MembershipLevel
}

type MembershipAssignment struct {
	ID                int64
	UserID            int64
	MembershipLevelID int64
	StartsAt          time.Time
	ExpiresAt         *time.Time
	Status            string
	Source            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Level             *MembershipLevel
	User              *User
}

type MembershipManagedKey struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"user_id"`
	GroupID           int64     `json:"group_id"`
	APIKeyID          int64     `json:"api_key_id"`
	MembershipLevelID int64     `json:"membership_level_id"`
	Status            string    `json:"status"`
	DisabledReason    string    `json:"disabled_reason"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Group             *Group    `json:"group,omitempty"`
	APIKey            *APIKey   `json:"api_key,omitempty"`
}

type MembershipSummary struct {
	Level       *MembershipLevel       `json:"level"`
	StartsAt    time.Time              `json:"starts_at"`
	ExpiresAt   *time.Time             `json:"expires_at"`
	Levels      []MembershipLevel      `json:"levels"`
	Groups      []Group                `json:"groups"`
	ManagedKeys []MembershipManagedKey `json:"managed_keys"`
}

type MembershipLevelInput struct {
	Name         string
	Code         string
	Color        string
	DiscountRate float64
	Enabled      bool
	IsDefault    bool
	SortOrder    int
	Description  string
	GroupIDs     []int64
}

type AssignMembershipInput struct {
	UserID    int64
	LevelID   int64
	ExpiresAt *time.Time
	Source    string
}

type MembershipRepository interface {
	ListMembershipLevels(ctx context.Context, includeDisabled bool) ([]MembershipLevel, error)
	GetMembershipLevel(ctx context.Context, id int64) (*MembershipLevel, error)
	GetDefaultMembershipLevel(ctx context.Context) (*MembershipLevel, error)
	CreateMembershipLevel(ctx context.Context, input MembershipLevelInput) (*MembershipLevel, error)
	UpdateMembershipLevel(ctx context.Context, id int64, input MembershipLevelInput) (*MembershipLevel, error)
	DisableMembershipLevel(ctx context.Context, id int64) error
	ListMembershipLevelsByGroup(ctx context.Context, groupID int64) ([]MembershipLevel, error)

	GetActiveUserMembership(ctx context.Context, userID int64) (*UserMembership, error)
	UpsertActiveUserMembership(ctx context.Context, userID, levelID int64, startsAt time.Time, expiresAt *time.Time, source string) (*UserMembership, error)
	MarkUserMembershipExpired(ctx context.Context, membershipID int64) error
	ListUserMembershipsByLevel(ctx context.Context, levelID int64) ([]UserMembership, error)
	ListExpiredActiveUserMemberships(ctx context.Context, now time.Time, limit int) ([]UserMembership, error)
	ListActiveUserMemberships(ctx context.Context, limit int) ([]UserMembership, error)
	ListActiveUserMembershipsAfterID(ctx context.Context, afterID int64, limit int) ([]UserMembership, error)

	ListManagedKeysByUser(ctx context.Context, userID int64) ([]MembershipManagedKey, error)
	GetManagedKeyByUserGroup(ctx context.Context, userID, groupID int64) (*MembershipManagedKey, error)
	GetManagedKeyByAPIKeyID(ctx context.Context, apiKeyID int64) (*MembershipManagedKey, error)
	UpsertManagedKey(ctx context.Context, key MembershipManagedKey) error
	SetManagedKeyStatus(ctx context.Context, userID, groupID int64, status, reason string, levelID int64) error
}

type MembershipBootstrapper interface {
	AssignDefaultMembership(ctx context.Context, userID int64) error
}

type MembershipService struct {
	repo              MembershipRepository
	userRepo          UserRepository
	groupRepo         GroupRepository
	userGroupRateRepo UserGroupRateRepository
	apiKeyService     *APIKeyService
}

func NewMembershipService(
	repo MembershipRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	userGroupRateRepo UserGroupRateRepository,
	apiKeyService *APIKeyService,
) *MembershipService {
	return &MembershipService{
		repo:              repo,
		userRepo:          userRepo,
		groupRepo:         groupRepo,
		userGroupRateRepo: userGroupRateRepo,
		apiKeyService:     apiKeyService,
	}
}

func (s *MembershipService) ListLevels(ctx context.Context, includeDisabled bool) ([]MembershipLevel, error) {
	return s.repo.ListMembershipLevels(ctx, includeDisabled)
}

func (s *MembershipService) GetLevel(ctx context.Context, id int64) (*MembershipLevel, error) {
	return s.repo.GetMembershipLevel(ctx, id)
}

func (s *MembershipService) CreateLevel(ctx context.Context, input MembershipLevelInput) (*MembershipLevel, error) {
	if input.DiscountRate < 0 {
		return nil, ErrMembershipInvalidDiscount
	}
	def, ok := fixedMembershipLevelByCode(strings.TrimSpace(input.Code))
	if !ok {
		return nil, ErrMembershipFixedLevelsOnly
	}
	applyFixedMembershipLevelDefinition(&input, def)
	color, err := normalizeMembershipLevelColor(input.Color)
	if err != nil {
		return nil, err
	}
	input.Color = color
	level, err := s.repo.CreateMembershipLevel(ctx, input)
	if err != nil {
		return nil, err
	}
	return level, nil
}

func (s *MembershipService) UpdateLevel(ctx context.Context, id int64, input MembershipLevelInput) (*MembershipLevel, error) {
	if input.DiscountRate < 0 {
		return nil, ErrMembershipInvalidDiscount
	}
	existing, err := s.repo.GetMembershipLevel(ctx, id)
	if err != nil {
		return nil, err
	}
	def, ok := fixedMembershipLevelByCode(existing.Code)
	if !ok {
		return nil, ErrMembershipFixedLevelsOnly
	}
	applyFixedMembershipLevelDefinition(&input, def)
	color, err := normalizeMembershipLevelColor(input.Color)
	if err != nil {
		return nil, err
	}
	input.Color = color
	level, err := s.repo.UpdateMembershipLevel(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := s.SyncMembershipLevel(ctx, id); err != nil {
		logger.LegacyPrintf("service.membership", "sync membership level failed: level_id=%d err=%v", id, err)
	}
	return level, nil
}

func (s *MembershipService) DisableLevel(ctx context.Context, id int64) error {
	existing, err := s.repo.GetMembershipLevel(ctx, id)
	if err != nil {
		return err
	}
	if _, ok := fixedMembershipLevelByCode(existing.Code); ok {
		return ErrMembershipFixedLevelsOnly
	}
	if err := s.repo.DisableMembershipLevel(ctx, id); err != nil {
		return err
	}
	return s.SyncMembershipLevel(ctx, id)
}

func (s *MembershipService) AssignDefaultMembership(ctx context.Context, userID int64) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil
	}
	if existing, err := s.repo.GetActiveUserMembership(ctx, userID); err == nil && existing != nil {
		return nil
	} else if err != nil && !errors.Is(err, ErrMembershipLevelNotFound) {
		return err
	}
	level, err := s.repo.GetDefaultMembershipLevel(ctx)
	if err != nil {
		if errors.Is(err, ErrMembershipDefaultNotFound) {
			logger.LegacyPrintf("service.membership", "default membership level not found: user_id=%d", userID)
			return nil
		}
		return err
	}
	_, err = s.repo.UpsertActiveUserMembership(ctx, userID, level.ID, time.Now(), nil, MembershipSourceSystem)
	if err != nil {
		return err
	}
	return s.SyncUserMembership(ctx, userID)
}

func (s *MembershipService) AssignMembership(ctx context.Context, input AssignMembershipInput) (*MembershipSummary, error) {
	if input.UserID <= 0 || input.LevelID <= 0 {
		return nil, ErrMembershipLevelNotFound
	}
	level, err := s.repo.GetMembershipLevel(ctx, input.LevelID)
	if err != nil {
		return nil, err
	}
	source := input.Source
	if source == "" {
		source = MembershipSourceAdmin
	}
	_, err = s.repo.UpsertActiveUserMembership(ctx, input.UserID, level.ID, time.Now(), input.ExpiresAt, source)
	if err != nil {
		return nil, err
	}
	if err := s.SyncUserMembership(ctx, input.UserID); err != nil {
		return nil, err
	}
	return s.GetUserMembership(ctx, input.UserID)
}

func (s *MembershipService) ListAssignments(ctx context.Context, limit int) ([]MembershipAssignment, error) {
	if limit <= 0 {
		limit = 200
	}
	memberships, err := s.repo.ListActiveUserMemberships(ctx, limit)
	if err != nil {
		return nil, err
	}
	assignments := make([]MembershipAssignment, 0, len(memberships))
	for _, membership := range memberships {
		assignment := MembershipAssignment{
			ID:                membership.ID,
			UserID:            membership.UserID,
			MembershipLevelID: membership.MembershipLevelID,
			StartsAt:          membership.StartsAt,
			ExpiresAt:         membership.ExpiresAt,
			Status:            membership.Status,
			Source:            membership.Source,
			CreatedAt:         membership.CreatedAt,
			UpdatedAt:         membership.UpdatedAt,
		}
		if level, err := s.repo.GetMembershipLevel(ctx, membership.MembershipLevelID); err == nil {
			assignment.Level = level
		} else {
			logger.LegacyPrintf("service.membership", "load assignment level failed: membership_id=%d level_id=%d err=%v", membership.ID, membership.MembershipLevelID, err)
		}
		if s.userRepo != nil {
			if user, err := s.userRepo.GetByID(ctx, membership.UserID); err == nil {
				assignment.User = user
			} else {
				logger.LegacyPrintf("service.membership", "load assignment user failed: membership_id=%d user_id=%d err=%v", membership.ID, membership.UserID, err)
			}
		}
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

func (s *MembershipService) GetUserMembership(ctx context.Context, userID int64) (*MembershipSummary, error) {
	membership, err := s.repo.GetActiveUserMembership(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrMembershipDefaultNotFound) {
			return nil, err
		}
		if err := s.AssignDefaultMembership(ctx, userID); err != nil {
			return nil, err
		}
		membership, err = s.repo.GetActiveUserMembership(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	level, err := s.repo.GetMembershipLevel(ctx, membership.MembershipLevelID)
	if err != nil {
		return nil, err
	}
	keys, err := s.repo.ListManagedKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	levels, err := s.repo.ListMembershipLevels(ctx, false)
	if err != nil {
		return nil, err
	}
	groups := make([]Group, 0, len(level.Groups))
	groups = append(groups, level.Groups...)
	if levels == nil {
		levels = make([]MembershipLevel, 0)
	}
	if keys == nil {
		keys = make([]MembershipManagedKey, 0)
	}
	return &MembershipSummary{
		Level:       level,
		StartsAt:    membership.StartsAt,
		ExpiresAt:   membership.ExpiresAt,
		Levels:      levels,
		Groups:      groups,
		ManagedKeys: keys,
	}, nil
}

func (s *MembershipService) SyncUserMembership(ctx context.Context, userID int64) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil
	}
	membership, err := s.repo.GetActiveUserMembership(ctx, userID)
	if err != nil {
		return s.AssignDefaultMembership(ctx, userID)
	}
	level, err := s.repo.GetMembershipLevel(ctx, membership.MembershipLevelID)
	if err != nil {
		if errors.Is(err, ErrMembershipLevelNotFound) {
			return s.AssignDefaultMembership(ctx, userID)
		}
		return err
	}
	if !level.Enabled {
		return s.expireToDefault(ctx, membership, ManagedKeyDisabledLevelDisabled)
	}
	if membership.ExpiresAt != nil && !time.Now().Before(*membership.ExpiresAt) {
		return s.expireToDefault(ctx, membership, ManagedKeyDisabledMembershipExpired)
	}

	managedKeys, err := s.repo.ListManagedKeysByUser(ctx, userID)
	if err != nil {
		return err
	}

	desired := make(map[int64]Group, len(level.Groups))
	for _, group := range level.Groups {
		desired[group.ID] = group
	}
	rateUpdates := make(map[int64]*float64)
	for _, group := range level.Groups {
		rate := group.RateMultiplier * level.DiscountRate
		rateUpdates[group.ID] = &rate
		if group.IsExclusive && s.userRepo != nil {
			if err := s.userRepo.AddGroupToAllowedGroups(ctx, userID, group.ID); err != nil {
				return fmt.Errorf("grant membership group access: %w", err)
			}
		}
		if err := s.EnsureManagedKey(ctx, userID, group.ID, level.ID); err != nil {
			return err
		}
	}

	for _, managed := range managedKeys {
		if _, ok := desired[managed.GroupID]; ok {
			continue
		}
		rateUpdates[managed.GroupID] = nil
		if managed.Group != nil && managed.Group.IsExclusive && s.userRepo != nil {
			if err := s.userRepo.RemoveGroupFromUserAllowedGroups(ctx, userID, managed.GroupID); err != nil {
				return fmt.Errorf("remove membership group access: %w", err)
			}
		}
		if err := s.DisableManagedKey(ctx, userID, managed.GroupID, ManagedKeyDisabledGroupRemoved); err != nil {
			return err
		}
	}

	if len(rateUpdates) > 0 && s.userGroupRateRepo != nil {
		if err := s.userGroupRateRepo.SyncUserGroupRates(ctx, userID, rateUpdates); err != nil {
			return fmt.Errorf("sync membership user rates: %w", err)
		}
	}
	return nil
}

func (s *MembershipService) SyncMembershipLevel(ctx context.Context, levelID int64) error {
	memberships, err := s.repo.ListUserMembershipsByLevel(ctx, levelID)
	if err != nil {
		return err
	}
	for _, membership := range memberships {
		if err := s.SyncUserMembership(ctx, membership.UserID); err != nil {
			logger.LegacyPrintf("service.membership", "sync user membership failed: user_id=%d level_id=%d err=%v", membership.UserID, levelID, err)
		}
	}
	return nil
}

func (s *MembershipService) SyncGroupRate(ctx context.Context, groupID int64) error {
	levels, err := s.repo.ListMembershipLevelsByGroup(ctx, groupID)
	if err != nil {
		return err
	}
	for _, level := range levels {
		if err := s.SyncMembershipLevel(ctx, level.ID); err != nil {
			logger.LegacyPrintf("service.membership", "sync group membership rate failed: group_id=%d level_id=%d err=%v", groupID, level.ID, err)
		}
	}
	return nil
}

func (s *MembershipService) ExpireMemberships(ctx context.Context) error {
	const batchSize = 500
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		expired, err := s.repo.ListExpiredActiveUserMemberships(ctx, time.Now(), batchSize)
		if err != nil {
			return err
		}
		if len(expired) == 0 {
			return nil
		}
		var firstErr error
		for _, membership := range expired {
			if err := s.expireToDefault(ctx, &membership, ManagedKeyDisabledMembershipExpired); err != nil {
				logger.LegacyPrintf("service.membership", "expire membership failed: user_id=%d membership_id=%d err=%v", membership.UserID, membership.ID, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		if firstErr != nil {
			return firstErr
		}
	}
}

func (s *MembershipService) RepairMembershipState(ctx context.Context) error {
	if err := s.ExpireMemberships(ctx); err != nil {
		return err
	}
	const batchSize = 5000
	var afterID int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		active, err := s.repo.ListActiveUserMembershipsAfterID(ctx, afterID, batchSize)
		if err != nil {
			return err
		}
		if len(active) == 0 {
			return nil
		}
		for _, membership := range active {
			if membership.ID > afterID {
				afterID = membership.ID
			}
			if err := s.SyncUserMembership(ctx, membership.UserID); err != nil {
				logger.LegacyPrintf("service.membership", "repair membership failed: user_id=%d membership_id=%d err=%v", membership.UserID, membership.ID, err)
			}
		}
	}
}

func (s *MembershipService) EnsureManagedKey(ctx context.Context, userID, groupID, levelID int64) error {
	if s.apiKeyService == nil {
		return nil
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetManagedKeyByUserGroup(ctx, userID, groupID)
	if err == nil && existing != nil {
		if shouldRestoreManagedAPIKey(existing) {
			status := StatusAPIKeyActive
			_, err := s.apiKeyService.Update(withManagedAPIKeyBypass(ctx), existing.APIKeyID, userID, UpdateAPIKeyRequest{Status: &status})
			if err != nil {
				return fmt.Errorf("restore membership api key: %w", err)
			}
			return s.repo.SetManagedKeyStatus(ctx, userID, groupID, ManagedKeyStatusActive, "", levelID)
		}
		if existing.APIKey != nil && existing.APIKey.Status == StatusAPIKeyActive {
			return s.repo.SetManagedKeyStatus(ctx, userID, groupID, ManagedKeyStatusActive, "", levelID)
		}
		return s.repo.SetManagedKeyStatus(ctx, userID, groupID, existing.Status, existing.DisabledReason, levelID)
	}
	if err != nil && !errors.Is(err, ErrAPIKeyNotFound) {
		return err
	}
	if group.IsExclusive && s.userRepo != nil {
		if err := s.userRepo.AddGroupToAllowedGroups(ctx, userID, groupID); err != nil {
			return fmt.Errorf("grant membership group access: %w", err)
		}
	}
	name := fmt.Sprintf("Membership Key - %s", group.Name)
	key, err := s.apiKeyService.Create(withManagedAPIKeyBypass(ctx), userID, CreateAPIKeyRequest{
		Name:    name,
		GroupID: &groupID,
	})
	if err != nil {
		return fmt.Errorf("create membership api key: %w", err)
	}
	return s.repo.UpsertManagedKey(ctx, MembershipManagedKey{
		UserID:            userID,
		GroupID:           groupID,
		APIKeyID:          key.ID,
		MembershipLevelID: levelID,
		Status:            ManagedKeyStatusActive,
	})
}

func (s *MembershipService) DisableManagedKey(ctx context.Context, userID, groupID int64, reason string) error {
	managed, err := s.repo.GetManagedKeyByUserGroup(ctx, userID, groupID)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return nil
		}
		return err
	}
	status := StatusAPIKeyDisabled
	if s.apiKeyService != nil {
		if _, err := s.apiKeyService.Update(withManagedAPIKeyBypass(ctx), managed.APIKeyID, userID, UpdateAPIKeyRequest{Status: &status}); err != nil {
			return fmt.Errorf("disable membership api key: %w", err)
		}
	}
	return s.repo.SetManagedKeyStatus(ctx, userID, groupID, ManagedKeyStatusDisabled, reason, managed.MembershipLevelID)
}

func (s *MembershipService) RestoreManagedKey(ctx context.Context, userID, groupID, levelID int64) error {
	return s.EnsureManagedKey(ctx, userID, groupID, levelID)
}

func shouldRestoreManagedAPIKey(key *MembershipManagedKey) bool {
	if key == nil || key.APIKey == nil {
		return false
	}
	if key.APIKey.Status != StatusAPIKeyDisabled || key.Status != ManagedKeyStatusDisabled {
		return false
	}
	return isMembershipManagedKeyDisabledReason(key.DisabledReason)
}

func isMembershipManagedKeyDisabledReason(reason string) bool {
	switch reason {
	case ManagedKeyDisabledMembershipExpired,
		ManagedKeyDisabledGroupRemoved,
		ManagedKeyDisabledLevelDisabled,
		ManagedKeyDisabledRepairDisabled:
		return true
	default:
		return false
	}
}

func (s *MembershipService) GetManagedKeyByAPIKeyID(ctx context.Context, apiKeyID int64) (*MembershipManagedKey, error) {
	return s.repo.GetManagedKeyByAPIKeyID(ctx, apiKeyID)
}

func (s *MembershipService) expireToDefault(ctx context.Context, membership *UserMembership, reason string) error {
	if membership == nil {
		return nil
	}
	if membership.ID > 0 {
		if err := s.repo.MarkUserMembershipExpired(ctx, membership.ID); err != nil {
			return err
		}
	}
	managedKeys, err := s.repo.ListManagedKeysByUser(ctx, membership.UserID)
	if err != nil {
		return err
	}
	for _, managed := range managedKeys {
		if err := s.DisableManagedKey(ctx, membership.UserID, managed.GroupID, reason); err != nil {
			logger.LegacyPrintf("service.membership", "disable expired membership key failed: user_id=%d group_id=%d err=%v", membership.UserID, managed.GroupID, err)
		}
		if managed.Group != nil && managed.Group.IsExclusive && s.userRepo != nil {
			_ = s.userRepo.RemoveGroupFromUserAllowedGroups(ctx, membership.UserID, managed.GroupID)
		}
	}
	if s.userGroupRateRepo != nil {
		clearRates := make(map[int64]*float64, len(managedKeys))
		for _, managed := range managedKeys {
			clearRates[managed.GroupID] = nil
		}
		if len(clearRates) > 0 {
			if err := s.userGroupRateRepo.SyncUserGroupRates(ctx, membership.UserID, clearRates); err != nil {
				return err
			}
		}
	}
	return s.AssignDefaultMembership(ctx, membership.UserID)
}

func normalizeMembershipLevelColor(color string) (string, error) {
	color = strings.TrimSpace(color)
	if color == "" {
		return defaultMembershipLevelColor, nil
	}
	if !membershipLevelColorPattern.MatchString(color) {
		return "", ErrMembershipInvalidColor
	}
	return strings.ToLower(color), nil
}

type managedAPIKeyBypassContextKey struct{}

func withManagedAPIKeyBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, managedAPIKeyBypassContextKey{}, true)
}

func isManagedAPIKeyBypass(ctx context.Context) bool {
	v, _ := ctx.Value(managedAPIKeyBypassContextKey{}).(bool)
	return v
}

type MembershipExpiryService struct {
	membershipService *MembershipService
	interval          time.Duration
	repairInterval    time.Duration
	lastRepair        time.Time
	stopCh            chan struct{}
}

func NewMembershipExpiryService(membershipService *MembershipService, interval time.Duration) *MembershipExpiryService {
	return &MembershipExpiryService{
		membershipService: membershipService,
		interval:          interval,
		repairInterval:    24 * time.Hour,
		stopCh:            make(chan struct{}),
	}
}

func (s *MembershipExpiryService) Start() {
	if s == nil || s.membershipService == nil || s.interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *MembershipExpiryService) Stop() {
	if s == nil || s.stopCh == nil {
		return
	}
	close(s.stopCh)
}

func (s *MembershipExpiryService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if s.lastRepair.IsZero() || time.Since(s.lastRepair) >= s.repairInterval {
		if err := s.membershipService.RepairMembershipState(ctx); err != nil {
			log.Printf("[MembershipExpiry] repair membership state failed: %v", err)
		} else {
			s.lastRepair = time.Now()
		}
		return
	}
	if err := s.membershipService.ExpireMemberships(ctx); err != nil {
		log.Printf("[MembershipExpiry] expire memberships failed: %v", err)
	}
}
