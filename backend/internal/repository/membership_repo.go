package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type membershipRepository struct {
	sql sqlExecutor
}

func NewMembershipRepository(sqlDB *sql.DB) service.MembershipRepository {
	return &membershipRepository{sql: sqlDB}
}

func (r *membershipRepository) withTx(ctx context.Context, fn func(sqlExecutor) error) error {
	beginner, ok := r.sql.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return fn(r.sql)
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *membershipRepository) ListMembershipLevels(ctx context.Context, includeDisabled bool) ([]service.MembershipLevel, error) {
	query := `
		SELECT id, name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at
		FROM membership_levels
	`
	var args []any
	if !includeDisabled {
		query += " WHERE enabled = TRUE"
	}
	query += " ORDER BY sort_order ASC, id ASC"
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var levels []service.MembershipLevel
	for rows.Next() {
		level, err := scanMembershipLevel(rows)
		if err != nil {
			return nil, err
		}
		if err := r.loadLevelGroups(ctx, level); err != nil {
			return nil, err
		}
		levels = append(levels, *level)
	}
	return levels, rows.Err()
}

func (r *membershipRepository) GetMembershipLevel(ctx context.Context, id int64) (*service.MembershipLevel, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at
		FROM membership_levels
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrMembershipLevelNotFound
	}
	level, err := scanMembershipLevel(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadLevelGroups(ctx, level); err != nil {
		return nil, err
	}
	return level, nil
}

func (r *membershipRepository) GetDefaultMembershipLevel(ctx context.Context) (*service.MembershipLevel, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at
		FROM membership_levels
		WHERE is_default = TRUE AND enabled = TRUE
		ORDER BY id ASC
		LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrMembershipDefaultNotFound
	}
	level, err := scanMembershipLevel(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadLevelGroups(ctx, level); err != nil {
		return nil, err
	}
	return level, nil
}

func (r *membershipRepository) CreateMembershipLevel(ctx context.Context, input service.MembershipLevelInput) (*service.MembershipLevel, error) {
	var id int64
	err := r.withTx(ctx, func(q sqlExecutor) error {
		if input.IsDefault {
			if _, err := q.ExecContext(ctx, `UPDATE membership_levels SET is_default = FALSE, updated_at = NOW() WHERE is_default = TRUE`); err != nil {
				return err
			}
		}
		if err := scanSingleRow(ctx, q, `
			INSERT INTO membership_levels (name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			RETURNING id
		`, []any{input.Name, input.Code, input.Color, input.DiscountRate, input.Enabled, input.IsDefault, input.SortOrder, input.Description}, &id); err != nil {
			return err
		}
		return replaceLevelGroups(ctx, q, id, input.GroupIDs)
	})
	if err != nil {
		return nil, err
	}
	return r.GetMembershipLevel(ctx, id)
}

func (r *membershipRepository) UpdateMembershipLevel(ctx context.Context, id int64, input service.MembershipLevelInput) (*service.MembershipLevel, error) {
	err := r.withTx(ctx, func(q sqlExecutor) error {
		if input.IsDefault {
			if _, err := q.ExecContext(ctx, `UPDATE membership_levels SET is_default = FALSE, updated_at = NOW() WHERE is_default = TRUE AND id <> $1`, id); err != nil {
				return err
			}
		}
		result, err := q.ExecContext(ctx, `
			UPDATE membership_levels
			SET name = $2,
				code = $3,
				color = $4,
				discount_rate = $5,
				enabled = $6,
				is_default = $7,
				sort_order = $8,
				description = $9,
				updated_at = NOW()
			WHERE id = $1
		`, id, input.Name, input.Code, input.Color, input.DiscountRate, input.Enabled, input.IsDefault, input.SortOrder, input.Description)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return service.ErrMembershipLevelNotFound
		}
		return replaceLevelGroups(ctx, q, id, input.GroupIDs)
	})
	if err != nil {
		return nil, err
	}
	return r.GetMembershipLevel(ctx, id)
}

func (r *membershipRepository) DisableMembershipLevel(ctx context.Context, id int64) error {
	result, err := r.sql.ExecContext(ctx, `
		UPDATE membership_levels
		SET enabled = FALSE, is_default = FALSE, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return service.ErrMembershipLevelNotFound
	}
	return nil
}

func (r *membershipRepository) ListMembershipLevelsByGroup(ctx context.Context, groupID int64) ([]service.MembershipLevel, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT ml.id, ml.name, ml.code, ml.color, ml.discount_rate, ml.enabled, ml.is_default, ml.sort_order, ml.description, ml.created_at, ml.updated_at
		FROM membership_levels ml
		JOIN membership_level_groups mlg ON mlg.membership_level_id = ml.id
		WHERE mlg.group_id = $1
		ORDER BY ml.sort_order ASC, ml.id ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var levels []service.MembershipLevel
	for rows.Next() {
		level, err := scanMembershipLevel(rows)
		if err != nil {
			return nil, err
		}
		if err := r.loadLevelGroups(ctx, level); err != nil {
			return nil, err
		}
		levels = append(levels, *level)
	}
	return levels, rows.Err()
}

func (r *membershipRepository) GetActiveUserMembership(ctx context.Context, userID int64) (*service.UserMembership, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, user_id, membership_level_id, starts_at, expires_at, status, source, created_at, updated_at
		FROM user_memberships
		WHERE user_id = $1 AND status = 'active'
		LIMIT 1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrMembershipLevelNotFound
	}
	membership, err := scanUserMembership(rows)
	if err != nil {
		return nil, err
	}
	return membership, rows.Err()
}

func (r *membershipRepository) UpsertActiveUserMembership(ctx context.Context, userID, levelID int64, startsAt time.Time, expiresAt *time.Time, source string) (*service.UserMembership, error) {
	result, err := r.sql.ExecContext(ctx, `
		UPDATE user_memberships
		SET membership_level_id = $2,
			starts_at = $3,
			expires_at = $4,
			source = $5,
			updated_at = NOW()
		WHERE user_id = $1 AND status = 'active'
	`, userID, levelID, startsAt, expiresAt, source)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		if _, err := r.sql.ExecContext(ctx, `
			INSERT INTO user_memberships (user_id, membership_level_id, starts_at, expires_at, status, source, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'active', $5, NOW(), NOW())
		`, userID, levelID, startsAt, expiresAt, source); err != nil {
			return nil, err
		}
	}
	return r.GetActiveUserMembership(ctx, userID)
}

func (r *membershipRepository) MarkUserMembershipExpired(ctx context.Context, membershipID int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE user_memberships
		SET status = 'expired', updated_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, membershipID)
	return err
}

func (r *membershipRepository) ListUserMembershipsByLevel(ctx context.Context, levelID int64) ([]service.UserMembership, error) {
	return r.listUserMemberships(ctx, `
		SELECT id, user_id, membership_level_id, starts_at, expires_at, status, source, created_at, updated_at
		FROM user_memberships
		WHERE membership_level_id = $1 AND status = 'active'
		ORDER BY id ASC
	`, levelID)
}

func (r *membershipRepository) ListExpiredActiveUserMemberships(ctx context.Context, now time.Time, limit int) ([]service.UserMembership, error) {
	if limit <= 0 {
		limit = 500
	}
	return r.listUserMemberships(ctx, `
		SELECT id, user_id, membership_level_id, starts_at, expires_at, status, source, created_at, updated_at
		FROM user_memberships
		WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at <= $1
		ORDER BY expires_at ASC, id ASC
		LIMIT $2
	`, now, limit)
}

func (r *membershipRepository) ListActiveUserMemberships(ctx context.Context, limit int) ([]service.UserMembership, error) {
	if limit <= 0 {
		limit = 5000
	}
	return r.listUserMemberships(ctx, `
		SELECT id, user_id, membership_level_id, starts_at, expires_at, status, source, created_at, updated_at
		FROM user_memberships
		WHERE status = 'active'
		ORDER BY id ASC
		LIMIT $1
	`, limit)
}

func (r *membershipRepository) ListActiveUserMembershipsAfterID(ctx context.Context, afterID int64, limit int) ([]service.UserMembership, error) {
	if limit <= 0 {
		limit = 5000
	}
	return r.listUserMemberships(ctx, `
		SELECT id, user_id, membership_level_id, starts_at, expires_at, status, source, created_at, updated_at
		FROM user_memberships
		WHERE status = 'active' AND id > $1
		ORDER BY id ASC
		LIMIT $2
	`, afterID, limit)
}

func (r *membershipRepository) ListManagedKeysByUser(ctx context.Context, userID int64) ([]service.MembershipManagedKey, error) {
	rows, err := r.sql.QueryContext(ctx, managedKeysSelectSQL()+`
		WHERE mk.user_id = $1
		ORDER BY g.sort_order ASC, g.id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var keys []service.MembershipManagedKey
	for rows.Next() {
		key, err := scanManagedKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, *key)
	}
	return keys, rows.Err()
}

func (r *membershipRepository) GetManagedKeyByUserGroup(ctx context.Context, userID, groupID int64) (*service.MembershipManagedKey, error) {
	rows, err := r.sql.QueryContext(ctx, managedKeysSelectSQL()+`
		WHERE mk.user_id = $1 AND mk.group_id = $2
		LIMIT 1
	`, userID, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrAPIKeyNotFound
	}
	key, err := scanManagedKey(rows)
	if err != nil {
		return nil, err
	}
	return key, rows.Err()
}

func (r *membershipRepository) GetManagedKeyByAPIKeyID(ctx context.Context, apiKeyID int64) (*service.MembershipManagedKey, error) {
	rows, err := r.sql.QueryContext(ctx, managedKeysSelectSQL()+`
		WHERE mk.api_key_id = $1
		LIMIT 1
	`, apiKeyID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, service.ErrAPIKeyNotFound
	}
	key, err := scanManagedKey(rows)
	if err != nil {
		return nil, err
	}
	return key, rows.Err()
}

func (r *membershipRepository) UpsertManagedKey(ctx context.Context, key service.MembershipManagedKey) error {
	_, err := r.sql.ExecContext(ctx, `
		INSERT INTO membership_managed_keys (
			user_id, group_id, api_key_id, membership_level_id, status, disabled_reason, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET
			api_key_id = EXCLUDED.api_key_id,
			membership_level_id = EXCLUDED.membership_level_id,
			status = EXCLUDED.status,
			disabled_reason = EXCLUDED.disabled_reason,
			updated_at = NOW()
	`, key.UserID, key.GroupID, key.APIKeyID, key.MembershipLevelID, key.Status, key.DisabledReason)
	return err
}

func (r *membershipRepository) SetManagedKeyStatus(ctx context.Context, userID, groupID int64, status, reason string, levelID int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE membership_managed_keys
		SET status = $3,
			disabled_reason = $4,
			membership_level_id = CASE WHEN $5 > 0 THEN $5 ELSE membership_level_id END,
			updated_at = NOW()
		WHERE user_id = $1 AND group_id = $2
	`, userID, groupID, status, reason, levelID)
	return err
}

func replaceLevelGroups(ctx context.Context, q sqlExecutor, levelID int64, groupIDs []int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM membership_level_groups WHERE membership_level_id = $1`, levelID); err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return nil
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO membership_level_groups (membership_level_id, group_id, created_at)
		SELECT $1, unnest($2::bigint[]), NOW()
		ON CONFLICT (membership_level_id, group_id) DO NOTHING
	`, levelID, pq.Array(groupIDs))
	return err
}

func (r *membershipRepository) loadLevelGroups(ctx context.Context, level *service.MembershipLevel) error {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT g.id, g.name, COALESCE(g.description, ''), g.platform, g.rate_multiplier, g.is_exclusive,
		       g.status, g.subscription_type, g.sort_order, g.created_at, g.updated_at
		FROM membership_level_groups mlg
		JOIN groups g ON g.id = mlg.group_id
		WHERE mlg.membership_level_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.sort_order ASC, g.id ASC
	`, level.ID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		group := service.Group{}
		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.Platform,
			&group.RateMultiplier,
			&group.IsExclusive,
			&group.Status,
			&group.SubscriptionType,
			&group.SortOrder,
			&group.CreatedAt,
			&group.UpdatedAt,
		); err != nil {
			return err
		}
		group.Hydrated = true
		level.Groups = append(level.Groups, group)
	}
	return rows.Err()
}

func (r *membershipRepository) listUserMemberships(ctx context.Context, query string, args ...any) ([]service.UserMembership, error) {
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var memberships []service.UserMembership
	for rows.Next() {
		membership, err := scanUserMembership(rows)
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, *membership)
	}
	return memberships, rows.Err()
}

func scanMembershipLevel(rows interface {
	Scan(dest ...any) error
}) (*service.MembershipLevel, error) {
	level := &service.MembershipLevel{}
	err := rows.Scan(
		&level.ID,
		&level.Name,
		&level.Code,
		&level.Color,
		&level.DiscountRate,
		&level.Enabled,
		&level.IsDefault,
		&level.SortOrder,
		&level.Description,
		&level.CreatedAt,
		&level.UpdatedAt,
	)
	return level, err
}

func scanUserMembership(rows interface {
	Scan(dest ...any) error
}) (*service.UserMembership, error) {
	membership := &service.UserMembership{}
	err := rows.Scan(
		&membership.ID,
		&membership.UserID,
		&membership.MembershipLevelID,
		&membership.StartsAt,
		&membership.ExpiresAt,
		&membership.Status,
		&membership.Source,
		&membership.CreatedAt,
		&membership.UpdatedAt,
	)
	return membership, err
}

func managedKeysSelectSQL() string {
	return `
		SELECT
			mk.id, mk.user_id, mk.group_id, mk.api_key_id, mk.membership_level_id,
			mk.status, mk.disabled_reason, mk.created_at, mk.updated_at,
			g.id, g.name, COALESCE(g.description, ''), g.platform, g.rate_multiplier, g.is_exclusive,
			g.status, g.subscription_type, g.sort_order, g.created_at, g.updated_at,
			ak.id, ak.user_id, ak.key, ak.name, ak.status, ak.group_id, ak.created_at, ak.updated_at
		FROM membership_managed_keys mk
		LEFT JOIN groups g ON g.id = mk.group_id
		LEFT JOIN api_keys ak ON ak.id = mk.api_key_id AND ak.deleted_at IS NULL
	`
}

func scanManagedKey(rows interface {
	Scan(dest ...any) error
}) (*service.MembershipManagedKey, error) {
	key := &service.MembershipManagedKey{}
	group := &service.Group{}
	apiKey := &service.APIKey{}
	var apiKeyID sql.NullInt64
	var apiKeyUserID sql.NullInt64
	var apiKeyValue sql.NullString
	var apiKeyName sql.NullString
	var apiKeyStatus sql.NullString
	var apiKeyGroupID sql.NullInt64
	var apiKeyCreatedAt sql.NullTime
	var apiKeyUpdatedAt sql.NullTime
	err := rows.Scan(
		&key.ID,
		&key.UserID,
		&key.GroupID,
		&key.APIKeyID,
		&key.MembershipLevelID,
		&key.Status,
		&key.DisabledReason,
		&key.CreatedAt,
		&key.UpdatedAt,
		&group.ID,
		&group.Name,
		&group.Description,
		&group.Platform,
		&group.RateMultiplier,
		&group.IsExclusive,
		&group.Status,
		&group.SubscriptionType,
		&group.SortOrder,
		&group.CreatedAt,
		&group.UpdatedAt,
		&apiKeyID,
		&apiKeyUserID,
		&apiKeyValue,
		&apiKeyName,
		&apiKeyStatus,
		&apiKeyGroupID,
		&apiKeyCreatedAt,
		&apiKeyUpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	group.Hydrated = group.ID > 0
	if group.ID > 0 {
		key.Group = group
	}
	if apiKeyID.Valid {
		apiKey.ID = apiKeyID.Int64
		if apiKeyUserID.Valid {
			apiKey.UserID = apiKeyUserID.Int64
		}
		apiKey.Key = apiKeyValue.String
		apiKey.Name = apiKeyName.String
		apiKey.Status = apiKeyStatus.String
		if apiKeyGroupID.Valid {
			v := apiKeyGroupID.Int64
			apiKey.GroupID = &v
		}
		if apiKeyCreatedAt.Valid {
			apiKey.CreatedAt = apiKeyCreatedAt.Time
		}
		if apiKeyUpdatedAt.Valid {
			apiKey.UpdatedAt = apiKeyUpdatedAt.Time
		}
		key.APIKey = apiKey
	}
	return key, nil
}
