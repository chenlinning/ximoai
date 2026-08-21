-- Remove the brass membership tier and clean unavailable membership-managed API keys.

INSERT INTO membership_levels (
    name, code, color, discount_rate, enabled, is_default, sort_order, description, created_at, updated_at
)
VALUES (
    '白银会员', 'silver', '#94a3b8', 1, TRUE, FALSE, 10, '系统固定白银会员等级', NOW(), NOW()
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    color = EXCLUDED.color,
    enabled = TRUE,
    sort_order = EXCLUDED.sort_order,
    description = EXCLUDED.description,
    updated_at = NOW();

WITH silver AS (
    SELECT id FROM membership_levels WHERE code = 'silver'
), bronze AS (
    SELECT id FROM membership_levels WHERE code = 'bronze'
)
INSERT INTO membership_level_groups (membership_level_id, group_id, created_at)
SELECT silver.id, mlg.group_id, NOW()
FROM silver
JOIN bronze ON TRUE
JOIN membership_level_groups mlg ON mlg.membership_level_id = bronze.id
ON CONFLICT (membership_level_id, group_id) DO NOTHING;

WITH silver AS (
    SELECT id FROM membership_levels WHERE code = 'silver'
), bronze AS (
    SELECT id FROM membership_levels WHERE code = 'bronze'
)
UPDATE user_memberships um
SET membership_level_id = silver.id,
    updated_at = NOW()
FROM silver, bronze
WHERE um.membership_level_id = bronze.id;

WITH silver AS (
    SELECT id FROM membership_levels WHERE code = 'silver'
), bronze AS (
    SELECT id FROM membership_levels WHERE code = 'bronze'
)
UPDATE membership_managed_keys mk
SET membership_level_id = silver.id,
    updated_at = NOW()
FROM silver, bronze
WHERE mk.membership_level_id = bronze.id;

UPDATE membership_levels
SET is_default = FALSE,
    updated_at = NOW()
WHERE is_default = TRUE;

UPDATE membership_levels
SET name = '白银会员',
    color = '#94a3b8',
    enabled = TRUE,
    is_default = TRUE,
    sort_order = 10,
    description = '系统固定白银会员等级',
    updated_at = NOW()
WHERE code = 'silver';

UPDATE membership_levels
SET sort_order = CASE code
        WHEN 'gold' THEN 20
        WHEN 'platinum' THEN 30
        WHEN 'diamond' THEN 40
        ELSE sort_order
    END,
    updated_at = NOW()
WHERE code IN ('gold', 'platinum', 'diamond');

DELETE FROM membership_levels
WHERE code = 'bronze';

UPDATE api_keys ak
SET key = '__deleted__' || ak.id || '__membership_cleanup',
    deleted_at = NOW(),
    updated_at = NOW()
FROM membership_managed_keys mk
LEFT JOIN groups g ON g.id = mk.group_id
WHERE ak.id = mk.api_key_id
  AND ak.deleted_at IS NULL
  AND (
      mk.status = 'disabled'
      OR ak.status <> 'active'
      OR g.id IS NULL
      OR g.deleted_at IS NOT NULL
      OR g.status <> 'active'
  );

DELETE FROM membership_managed_keys mk
WHERE mk.status = 'disabled'
   OR NOT EXISTS (
       SELECT 1
       FROM api_keys ak
       WHERE ak.id = mk.api_key_id
         AND ak.deleted_at IS NULL
         AND ak.status = 'active'
   )
   OR NOT EXISTS (
       SELECT 1
       FROM groups g
       WHERE g.id = mk.group_id
         AND g.deleted_at IS NULL
         AND g.status = 'active'
   );

UPDATE api_keys ak
SET name = LEFT('Membership Key - ' || g.name, 100),
    updated_at = NOW()
FROM membership_managed_keys mk
JOIN groups g ON g.id = mk.group_id
WHERE ak.id = mk.api_key_id
  AND ak.deleted_at IS NULL
  AND ak.status = 'active'
  AND ak.name IS DISTINCT FROM LEFT('Membership Key - ' || g.name, 100);
