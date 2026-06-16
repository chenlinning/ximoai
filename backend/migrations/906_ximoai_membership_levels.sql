-- XimoAI membership level system.

CREATE TABLE IF NOT EXISTS membership_levels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    discount_rate DOUBLE PRECISION NOT NULL DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT membership_levels_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT membership_levels_discount_rate_non_negative CHECK (discount_rate >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_membership_levels_code
    ON membership_levels (code);

CREATE UNIQUE INDEX IF NOT EXISTS idx_membership_levels_default
    ON membership_levels (is_default)
    WHERE is_default = TRUE;

CREATE TABLE IF NOT EXISTS membership_level_groups (
    id BIGSERIAL PRIMARY KEY,
    membership_level_id BIGINT NOT NULL REFERENCES membership_levels(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT membership_level_groups_unique UNIQUE (membership_level_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_membership_level_groups_group_id
    ON membership_level_groups (group_id);

CREATE TABLE IF NOT EXISTS user_memberships (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    membership_level_id BIGINT NOT NULL REFERENCES membership_levels(id),
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NULL,
    status TEXT NOT NULL DEFAULT 'active',
    source TEXT NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_memberships_status_check CHECK (status IN ('active', 'expired', 'disabled')),
    CONSTRAINT user_memberships_source_check CHECK (source IN ('system', 'admin', 'purchase'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_memberships_one_active
    ON user_memberships (user_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_user_memberships_level_status
    ON user_memberships (membership_level_id, status);

CREATE INDEX IF NOT EXISTS idx_user_memberships_expires_at
    ON user_memberships (expires_at)
    WHERE status = 'active' AND expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS membership_managed_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    membership_level_id BIGINT NOT NULL REFERENCES membership_levels(id),
    status TEXT NOT NULL DEFAULT 'active',
    disabled_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT membership_managed_keys_status_check CHECK (status IN ('active', 'disabled')),
    CONSTRAINT membership_managed_keys_disabled_reason_check CHECK (
        disabled_reason = '' OR disabled_reason IN (
            'membership_expired',
            'membership_group_removed',
            'membership_level_disabled',
            'repair_disabled'
        )
    ),
    CONSTRAINT membership_managed_keys_user_group_unique UNIQUE (user_id, group_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_membership_managed_keys_api_key_id
    ON membership_managed_keys (api_key_id);

CREATE INDEX IF NOT EXISTS idx_membership_managed_keys_user_id
    ON membership_managed_keys (user_id);

INSERT INTO membership_levels (name, code, discount_rate, enabled, is_default, sort_order, description)
VALUES ('默认会员', 'default', 1, TRUE, TRUE, 0, '系统默认会员等级')
ON CONFLICT (code) DO UPDATE SET
    enabled = TRUE,
    is_default = TRUE,
    updated_at = NOW();
