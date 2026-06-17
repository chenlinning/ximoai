-- Rename the first fixed membership tier to brass and apply its fixed color.

UPDATE membership_levels
SET name = '黄铜会员',
    color = '#b7791f',
    description = '系统固定黄铜会员等级',
    updated_at = NOW()
WHERE code = 'bronze';
