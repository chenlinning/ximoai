ALTER TABLE platforms
    ADD COLUMN IF NOT EXISTS kind VARCHAR(32) NOT NULL DEFAULT '';

UPDATE platforms
SET kind = CASE slug
    WHEN 'grok-video' THEN 'grok_video'
    WHEN 'openai-audio' THEN 'openai_audio'
    WHEN 'kling_audio' THEN 'kling_audio'
    ELSE kind
END,
updated_at = NOW()
WHERE slug IN ('grok-video', 'openai-audio', 'kling_audio');

CREATE UNIQUE INDEX IF NOT EXISTS idx_platforms_kind_unique
    ON platforms(kind)
    WHERE kind <> '';

UPDATE accounts
SET credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{platform_kind}',
        to_jsonb(CASE platform
            WHEN 'grok-video' THEN 'grok_video'
            WHEN 'openai-audio' THEN 'openai_audio'
            WHEN 'kling_audio' THEN 'kling_audio'
        END::text),
        TRUE
    ),
    updated_at = NOW()
WHERE platform IN ('grok-video', 'openai-audio', 'kling_audio');
