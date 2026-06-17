-- Add the platinum fixed membership tier and apply the fixed tier palette.

UPDATE membership_levels
SET name = '黄铜会员',
    color = '#b87333',
    description = '系统固定黄铜会员等级',
    sort_order = 10,
    enabled = TRUE,
    updated_at = NOW()
WHERE code = 'bronze';

UPDATE membership_levels
SET name = '白银会员',
    color = '#c0c7d1',
    description = '系统固定白银会员等级',
    sort_order = 20,
    enabled = TRUE,
    updated_at = NOW()
WHERE code = 'silver';

UPDATE membership_levels
SET name = '黄金会员',
    color = '#f2b705',
    description = '系统固定黄金会员等级',
    sort_order = 30,
    enabled = TRUE,
    updated_at = NOW()
WHERE code = 'gold';

INSERT INTO membership_levels (name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at)
VALUES ('铂金会员', 'platinum', '#d6b76a', 1, TRUE, FALSE, 40, '系统固定铂金会员等级', NOW(), NOW())
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    color = EXCLUDED.color,
    enabled = TRUE,
    sort_order = EXCLUDED.sort_order,
    description = EXCLUDED.description,
    updated_at = NOW();

UPDATE membership_levels
SET name = '钻石会员',
    color = '#0284c7',
    description = '系统固定钻石会员等级',
    sort_order = 50,
    enabled = TRUE,
    updated_at = NOW()
WHERE code = 'diamond';
