package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type platformRepository struct {
	db *sql.DB
}

func NewPlatformRepository(db *sql.DB) service.PlatformRepository {
	return &platformRepository{db: db}
}

func (r *platformRepository) List(ctx context.Context, includeDisabled bool) ([]service.Platform, error) {
	query := `
SELECT slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin, created_at, updated_at
FROM platforms`
	args := []any{}
	if !includeDisabled {
		query += ` WHERE enabled = TRUE`
	}
	query += ` ORDER BY builtin DESC, display_name ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []service.Platform
	for rows.Next() {
		platform, err := scanPlatform(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *platform)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *platformRepository) GetBySlug(ctx context.Context, slug string) (*service.Platform, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin, created_at, updated_at
FROM platforms
WHERE slug = $1`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPlatformNotFound
	}
	platform, err := scanPlatform(rows)
	if err != nil {
		return nil, err
	}
	return platform, rows.Err()
}

func (r *platformRepository) Create(ctx context.Context, platform *service.Platform) error {
	if platform == nil {
		return errors.New("platform is nil")
	}
	authModes, err := json.Marshal(platform.AuthModes)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(platform.Capabilities)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, FALSE)`,
		platform.Slug, platform.DisplayName, platform.Protocol, platform.BaseURL,
		string(authModes), string(capabilities), platform.Color, platform.Enabled)
	return err
}

func (r *platformRepository) Update(ctx context.Context, platform *service.Platform) error {
	if platform == nil {
		return errors.New("platform is nil")
	}
	authModes, err := json.Marshal(platform.AuthModes)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(platform.Capabilities)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE platforms
SET display_name = $2,
    protocol = $3,
    base_url = $4,
    auth_modes = $5::jsonb,
    capabilities = $6::jsonb,
    color = $7,
    enabled = $8,
    updated_at = NOW()
WHERE slug = $1`,
		platform.Slug, platform.DisplayName, platform.Protocol, platform.BaseURL,
		string(authModes), string(capabilities), platform.Color, platform.Enabled)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrPlatformNotFound
	}
	return nil
}

func (r *platformRepository) Rename(ctx context.Context, oldSlug string, platform *service.Platform) error {
	if platform == nil {
		return errors.New("platform is nil")
	}
	authModes, err := json.Marshal(platform.AuthModes)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(platform.Capabilities)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(ctx, `
UPDATE platforms
SET slug = $2,
    display_name = $3,
    protocol = $4,
    base_url = $5,
    auth_modes = $6::jsonb,
    capabilities = $7::jsonb,
    color = $8,
    enabled = $9,
    updated_at = NOW()
WHERE slug = $1`,
		oldSlug, platform.Slug, platform.DisplayName, platform.Protocol, platform.BaseURL,
		string(authModes), string(capabilities), platform.Color, platform.Enabled)
	if err != nil {
		return fmt.Errorf("rename platform: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrPlatformNotFound
	}
	if _, err = tx.ExecContext(ctx, `UPDATE accounts SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename account platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE groups SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename group platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE channel_model_pricing SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename channel pricing platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE channel_account_stats_model_pricing SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename account stats pricing platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE channels SET model_mapping = jsonb_set(model_mapping - $1, ARRAY[$2], model_mapping -> $1, true), updated_at = NOW() WHERE model_mapping ? $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename channel mapping platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE openai_video_jobs SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename video job platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE openai_video_characters SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename video character platforms: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE user_platform_quotas SET platform = $2, updated_at = NOW() WHERE platform = $1`, oldSlug, platform.Slug); err != nil {
		return fmt.Errorf("rename user platform quotas: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *platformRepository) Delete(ctx context.Context, slug string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM platforms WHERE slug = $1 AND builtin = FALSE`, slug)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrPlatformNotFound
	}
	return nil
}

func (r *platformRepository) Usage(ctx context.Context, slug string) (service.PlatformUsage, error) {
	var usage service.PlatformUsage
	err := r.db.QueryRowContext(ctx, `
SELECT
    (SELECT COUNT(*) FROM accounts WHERE platform = $1),
    (SELECT COUNT(*) FROM groups WHERE platform = $1),
    (SELECT COUNT(*) FROM channel_model_pricing WHERE platform = $1),
    (SELECT COUNT(*) FROM channel_account_stats_model_pricing WHERE platform = $1),
    (SELECT COUNT(*) FROM channels WHERE model_mapping ? $1)
`, slug).Scan(
		&usage.AccountCount,
		&usage.GroupCount,
		&usage.ChannelPricingCount,
		&usage.AccountStatsPricingCount,
		&usage.ChannelMappingCount,
	)
	if err != nil {
		return service.PlatformUsage{}, err
	}
	return usage, nil
}

type platformScanner interface {
	Scan(dest ...any) error
}

func scanPlatform(scanner platformScanner) (*service.Platform, error) {
	var p service.Platform
	var authModesRaw, capabilitiesRaw []byte
	if err := scanner.Scan(
		&p.Slug,
		&p.DisplayName,
		&p.Protocol,
		&p.BaseURL,
		&authModesRaw,
		&capabilitiesRaw,
		&p.Color,
		&p.Enabled,
		&p.Builtin,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrPlatformNotFound
		}
		return nil, err
	}
	_ = json.Unmarshal(authModesRaw, &p.AuthModes)
	_ = json.Unmarshal(capabilitiesRaw, &p.Capabilities)
	if p.AuthModes == nil {
		p.AuthModes = []string{}
	}
	if p.Capabilities == nil {
		p.Capabilities = []string{}
	}
	return &p, nil
}
