-- Align fixed membership level colors with the image badge palette.

UPDATE membership_levels
SET color = '#a15a2b',
    updated_at = NOW()
WHERE code = 'bronze';

UPDATE membership_levels
SET color = '#94a3b8',
    updated_at = NOW()
WHERE code = 'silver';

UPDATE membership_levels
SET color = '#d99a00',
    updated_at = NOW()
WHERE code = 'gold';

UPDATE membership_levels
SET color = '#b89a56',
    updated_at = NOW()
WHERE code = 'platinum';

UPDATE membership_levels
SET color = '#0ea5e9',
    updated_at = NOW()
WHERE code = 'diamond';
