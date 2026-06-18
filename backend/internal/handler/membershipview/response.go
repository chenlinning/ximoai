package membershipview

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Group struct {
	ID                      int64     `json:"id"`
	Name                    string    `json:"name"`
	Description             string    `json:"description"`
	Platform                string    `json:"platform"`
	RateMultiplier          float64   `json:"rate_multiplier"`
	EffectiveRateMultiplier *float64  `json:"effective_rate_multiplier,omitempty"`
	IsExclusive             bool      `json:"is_exclusive"`
	Status                  string    `json:"status"`
	SubscriptionType        string    `json:"subscription_type"`
	SortOrder               int       `json:"sort_order"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type Level struct {
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

type APIKey struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	KeySuffix string    `json:"key_suffix,omitempty"`
	MaskedKey string    `json:"masked_key,omitempty"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	GroupID   *int64    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ManagedKey struct {
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

type ExternalAPIKey struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Key       string    `json:"key"`
	KeySuffix string    `json:"key_suffix,omitempty"`
	MaskedKey string    `json:"masked_key,omitempty"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	GroupID   *int64    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExternalManagedKey struct {
	ID                int64           `json:"id"`
	UserID            int64           `json:"user_id"`
	GroupID           int64           `json:"group_id"`
	APIKeyID          int64           `json:"api_key_id"`
	MembershipLevelID int64           `json:"membership_level_id"`
	Status            string          `json:"status"`
	DisabledReason    string          `json:"disabled_reason"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	Group             *Group          `json:"group,omitempty"`
	APIKey            *ExternalAPIKey `json:"api_key,omitempty"`
}

type Summary struct {
	Level       *Level       `json:"level"`
	StartsAt    time.Time    `json:"starts_at"`
	ExpiresAt   *time.Time   `json:"expires_at"`
	Levels      []Level      `json:"levels"`
	Groups      []Group      `json:"groups"`
	ManagedKeys []ManagedKey `json:"managed_keys"`
}

type ExternalProfile struct {
	UserID     int64            `json:"user_id"`
	Membership *ExternalSummary `json:"membership"`
}

type ExternalSummary struct {
	Level       *Level               `json:"level"`
	StartsAt    time.Time            `json:"starts_at"`
	ExpiresAt   *time.Time           `json:"expires_at"`
	Levels      []Level              `json:"levels"`
	Groups      []Group              `json:"groups"`
	ManagedKeys []ExternalManagedKey `json:"managed_keys"`
}

type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Status   string `json:"status"`
}

type Assignment struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	MembershipLevelID int64      `json:"membership_level_id"`
	StartsAt          time.Time  `json:"starts_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	Status            string     `json:"status"`
	Source            string     `json:"source"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Level             *Level     `json:"level,omitempty"`
	User              *User      `json:"user,omitempty"`
}

func LevelFromService(level *service.MembershipLevel) *Level {
	if level == nil {
		return nil
	}
	discount := level.DiscountRate
	out := &Level{
		ID:           level.ID,
		Name:         level.Name,
		Code:         level.Code,
		Color:        level.Color,
		DiscountRate: level.DiscountRate,
		Enabled:      level.Enabled,
		IsDefault:    level.IsDefault,
		SortOrder:    level.SortOrder,
		Description:  level.Description,
		CreatedAt:    level.CreatedAt,
		UpdatedAt:    level.UpdatedAt,
		Groups:       make([]Group, 0, len(level.Groups)),
	}
	for _, group := range level.Groups {
		out.Groups = append(out.Groups, GroupFromService(&group, &discount))
	}
	return out
}

func LevelsFromService(levels []service.MembershipLevel) []Level {
	out := make([]Level, 0, len(levels))
	for i := range levels {
		if level := LevelFromService(&levels[i]); level != nil {
			out = append(out, *level)
		}
	}
	return out
}

func GroupFromService(group *service.Group, discount *float64) Group {
	if group == nil {
		return Group{}
	}
	out := Group{
		ID:               group.ID,
		Name:             group.Name,
		Description:      group.Description,
		Platform:         group.Platform,
		RateMultiplier:   group.RateMultiplier,
		IsExclusive:      group.IsExclusive,
		Status:           group.Status,
		SubscriptionType: group.SubscriptionType,
		SortOrder:        group.SortOrder,
		CreatedAt:        group.CreatedAt,
		UpdatedAt:        group.UpdatedAt,
	}
	if discount != nil {
		effective := group.RateMultiplier * *discount
		out.EffectiveRateMultiplier = &effective
	}
	return out
}

func SummaryFromService(summary *service.MembershipSummary) *Summary {
	if summary == nil {
		return nil
	}
	var discount *float64
	if summary.Level != nil {
		v := summary.Level.DiscountRate
		discount = &v
	}
	out := &Summary{
		Level:       LevelFromService(summary.Level),
		StartsAt:    summary.StartsAt,
		ExpiresAt:   summary.ExpiresAt,
		Levels:      LevelsFromService(summary.Levels),
		Groups:      make([]Group, 0, len(summary.Groups)),
		ManagedKeys: make([]ManagedKey, 0, len(summary.ManagedKeys)),
	}
	for _, group := range summary.Groups {
		out.Groups = append(out.Groups, GroupFromService(&group, discount))
	}
	for _, key := range summary.ManagedKeys {
		out.ManagedKeys = append(out.ManagedKeys, ManagedKeyFromService(&key, discount))
	}
	return out
}

func ExternalProfileFromService(userID int64, summary *service.MembershipSummary) *ExternalProfile {
	return &ExternalProfile{
		UserID:     userID,
		Membership: ExternalSummaryFromService(summary),
	}
}

func ExternalSummaryFromService(summary *service.MembershipSummary) *ExternalSummary {
	if summary == nil {
		return nil
	}
	var discount *float64
	if summary.Level != nil {
		v := summary.Level.DiscountRate
		discount = &v
	}
	out := &ExternalSummary{
		Level:       LevelFromService(summary.Level),
		StartsAt:    summary.StartsAt,
		ExpiresAt:   summary.ExpiresAt,
		Levels:      LevelsFromService(summary.Levels),
		Groups:      make([]Group, 0, len(summary.Groups)),
		ManagedKeys: make([]ExternalManagedKey, 0, len(summary.ManagedKeys)),
	}
	for _, group := range summary.Groups {
		out.Groups = append(out.Groups, GroupFromService(&group, discount))
	}
	for _, key := range summary.ManagedKeys {
		out.ManagedKeys = append(out.ManagedKeys, ExternalManagedKeyFromService(&key, discount))
	}
	return out
}

func ExternalManagedKeyFromService(key *service.MembershipManagedKey, discount *float64) ExternalManagedKey {
	out := ExternalManagedKey{
		ID:                key.ID,
		UserID:            key.UserID,
		GroupID:           key.GroupID,
		APIKeyID:          key.APIKeyID,
		MembershipLevelID: key.MembershipLevelID,
		Status:            key.Status,
		DisabledReason:    key.DisabledReason,
		CreatedAt:         key.CreatedAt,
		UpdatedAt:         key.UpdatedAt,
	}
	if key.Group != nil {
		group := GroupFromService(key.Group, discount)
		out.Group = &group
	}
	if key.APIKey != nil {
		out.APIKey = &ExternalAPIKey{
			ID:        key.APIKey.ID,
			UserID:    key.APIKey.UserID,
			Key:       key.APIKey.Key,
			KeySuffix: apiKeySuffix(key.APIKey.Key),
			MaskedKey: maskAPIKey(key.APIKey.Key),
			Name:      key.APIKey.Name,
			Status:    key.APIKey.Status,
			GroupID:   key.APIKey.GroupID,
			CreatedAt: key.APIKey.CreatedAt,
			UpdatedAt: key.APIKey.UpdatedAt,
		}
	}
	return out
}

func ManagedKeyFromService(key *service.MembershipManagedKey, discount *float64) ManagedKey {
	out := ManagedKey{
		ID:                key.ID,
		UserID:            key.UserID,
		GroupID:           key.GroupID,
		APIKeyID:          key.APIKeyID,
		MembershipLevelID: key.MembershipLevelID,
		Status:            key.Status,
		DisabledReason:    key.DisabledReason,
		CreatedAt:         key.CreatedAt,
		UpdatedAt:         key.UpdatedAt,
	}
	if key.Group != nil {
		group := GroupFromService(key.Group, discount)
		out.Group = &group
	}
	if key.APIKey != nil {
		out.APIKey = &APIKey{
			ID:        key.APIKey.ID,
			UserID:    key.APIKey.UserID,
			KeySuffix: apiKeySuffix(key.APIKey.Key),
			MaskedKey: maskAPIKey(key.APIKey.Key),
			Name:      key.APIKey.Name,
			Status:    key.APIKey.Status,
			GroupID:   key.APIKey.GroupID,
			CreatedAt: key.APIKey.CreatedAt,
			UpdatedAt: key.APIKey.UpdatedAt,
		}
	}
	return out
}

func apiKeySuffix(key string) string {
	if len(key) <= 5 {
		return key
	}
	return key[len(key)-5:]
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 5 {
		return key
	}
	return "sk-..." + apiKeySuffix(key)
}

func AssignmentFromService(assignment *service.MembershipAssignment) Assignment {
	out := Assignment{
		ID:                assignment.ID,
		UserID:            assignment.UserID,
		MembershipLevelID: assignment.MembershipLevelID,
		StartsAt:          assignment.StartsAt,
		ExpiresAt:         assignment.ExpiresAt,
		Status:            assignment.Status,
		Source:            assignment.Source,
		CreatedAt:         assignment.CreatedAt,
		UpdatedAt:         assignment.UpdatedAt,
		Level:             LevelFromService(assignment.Level),
	}
	if assignment.User != nil {
		out.User = &User{
			ID:       assignment.User.ID,
			Email:    assignment.User.Email,
			Username: assignment.User.Username,
			Status:   assignment.User.Status,
		}
	}
	return out
}

func AssignmentsFromService(assignments []service.MembershipAssignment) []Assignment {
	out := make([]Assignment, 0, len(assignments))
	for i := range assignments {
		out = append(out, AssignmentFromService(&assignments[i]))
	}
	return out
}
