-- XimoAI membership level display colors.

ALTER TABLE membership_levels
    ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '#4f46e5';

UPDATE membership_levels
SET color = '#4f46e5'
WHERE btrim(color) = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'membership_levels_color_format'
    ) THEN
        ALTER TABLE membership_levels
            ADD CONSTRAINT membership_levels_color_format
            CHECK (color ~ '^#[0-9A-Fa-f]{6}$');
    END IF;
END $$;
