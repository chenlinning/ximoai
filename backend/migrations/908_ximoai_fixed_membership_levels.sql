-- XimoAI fixed membership tiers.

UPDATE membership_levels
SET is_default = FALSE,
    updated_at = NOW()
WHERE is_default = TRUE;

INSERT INTO membership_levels (name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at)
VALUES
    ('青铜会员', 'bronze', '#a16207', 1, TRUE, TRUE, 10, '系统固定青铜会员等级', NOW(), NOW()),
    ('白银会员', 'silver', '#64748b', 1, TRUE, FALSE, 20, '系统固定白银会员等级', NOW(), NOW()),
    ('黄金会员', 'gold', '#d97706', 1, TRUE, FALSE, 30, '系统固定黄金会员等级', NOW(), NOW()),
    ('钻石会员', 'diamond', '#0891b2', 1, TRUE, FALSE, 40, '系统固定钻石会员等级', NOW(), NOW())
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    color = EXCLUDED.color,
    enabled = TRUE,
    is_default = EXCLUDED.is_default,
    sort_order = EXCLUDED.sort_order,
    description = EXCLUDED.description,
    updated_at = NOW();

WITH bronze AS (
    SELECT id FROM membership_levels WHERE code = 'bronze'
),
legacy_default AS (
    SELECT id FROM membership_levels WHERE code = 'default'
)
INSERT INTO membership_level_groups (membership_level_id, group_id, created_at)
SELECT bronze.id, mlg.group_id, NOW()
FROM bronze
JOIN legacy_default ON TRUE
JOIN membership_level_groups mlg ON mlg.membership_level_id = legacy_default.id
ON CONFLICT (membership_level_id, group_id) DO NOTHING;

WITH bronze AS (
    SELECT id FROM membership_levels WHERE code = 'bronze'
),
legacy_levels AS (
    SELECT id FROM membership_levels WHERE code NOT IN ('bronze', 'silver', 'gold', 'diamond')
)
UPDATE user_memberships
SET membership_level_id = (SELECT id FROM bronze),
    updated_at = NOW()
WHERE membership_level_id IN (SELECT id FROM legacy_levels);

WITH bronze AS (
    SELECT id FROM membership_levels WHERE code = 'bronze'
),
legacy_levels AS (
    SELECT id FROM membership_levels WHERE code NOT IN ('bronze', 'silver', 'gold', 'diamond')
)
UPDATE membership_managed_keys
SET membership_level_id = (SELECT id FROM bronze),
    updated_at = NOW()
WHERE membership_level_id IN (SELECT id FROM legacy_levels);

UPDATE membership_levels
SET enabled = FALSE,
    is_default = FALSE,
    updated_at = NOW()
WHERE code NOT IN ('bronze', 'silver', 'gold', 'diamond');
