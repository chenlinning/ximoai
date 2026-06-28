INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES
    ('grok-video', 'Grok-video', 'openai_compatible', '', '["apikey"]'::jsonb, '["videos"]'::jsonb, '#111827', TRUE, TRUE)
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    protocol = EXCLUDED.protocol,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    color = EXCLUDED.color,
    enabled = TRUE,
    builtin = TRUE,
    updated_at = NOW();

DO $$
DECLARE
    constraint_name text;
BEGIN
    SELECT con.conname
      INTO constraint_name
      FROM pg_constraint con
      JOIN pg_class rel ON rel.oid = con.conrelid
      JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
     WHERE rel.relname = 'user_platform_quotas'
       AND nsp.nspname = current_schema()
       AND con.contype = 'c'
       AND pg_get_constraintdef(con.oid) LIKE '%platform%'
     LIMIT 1;

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE user_platform_quotas DROP CONSTRAINT %I', constraint_name);
    END IF;

    ALTER TABLE user_platform_quotas
      ADD CONSTRAINT user_platform_quotas_platform_check
      CHECK (platform ~ '^[a-z0-9][a-z0-9_-]{1,62}[a-z0-9]$');
END $$;

UPDATE platforms
SET
    display_name = 'Grok',
    protocol = 'openai_compatible',
    auth_modes = '["oauth","apikey"]'::jsonb,
    capabilities = '["responses","chat_completions","videos"]'::jsonb,
    color = '#111827',
    enabled = TRUE,
    builtin = TRUE,
    updated_at = NOW()
WHERE slug = 'grok';

WITH video_only_grok_accounts AS (
    SELECT id
    FROM accounts
    WHERE platform = 'grok'
      AND type = 'apikey'
      AND COALESCE(credentials, '{}'::jsonb) ? 'api_key'
      AND NOT (COALESCE(credentials, '{}'::jsonb) ? 'access_token')
      AND NOT (COALESCE(credentials, '{}'::jsonb) ? 'refresh_token')
      AND (
        COALESCE(credentials, '{}'::jsonb) ? 'platform_protocol'
        OR NULLIF(TRIM(COALESCE(credentials ->> 'base_url', '')), '') IS NOT NULL
      )
      AND COALESCE(credentials ->> 'base_url', '') NOT IN ('https://api.x.ai', 'https://api.x.ai/v1')
)
UPDATE accounts
SET platform = 'grok-video',
    credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{platform_protocol}',
        to_jsonb('openai_compatible'::text),
        true
    ),
    updated_at = NOW()
WHERE id IN (SELECT id FROM video_only_grok_accounts);

UPDATE groups AS g
SET platform = 'grok-video',
    updated_at = NOW()
WHERE g.platform = 'grok'
  AND g.require_oauth_only = FALSE
  AND EXISTS (
      SELECT 1
      FROM account_groups ag
      JOIN accounts a ON a.id = ag.account_id
      WHERE ag.group_id = g.id
        AND a.platform = 'grok-video'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM account_groups ag
      JOIN accounts a ON a.id = ag.account_id
      WHERE ag.group_id = g.id
        AND a.platform = 'grok'
        AND a.type = 'oauth'
  );

UPDATE channel_model_pricing
SET platform = 'grok-video',
    updated_at = NOW()
WHERE platform = 'grok'
  AND billing_mode = 'video';

UPDATE channel_account_stats_model_pricing
SET platform = 'grok-video',
    updated_at = NOW()
WHERE platform = 'grok'
  AND billing_mode = 'video';

UPDATE channels
SET model_mapping = jsonb_set(model_mapping - 'grok', '{grok-video}', model_mapping -> 'grok', true),
    updated_at = NOW()
WHERE model_mapping ? 'grok'
  AND NOT (model_mapping ? 'grok-video');

UPDATE openai_video_jobs
SET platform = 'grok-video',
    updated_at = NOW()
WHERE platform = 'grok'
  AND account_id IN (SELECT id FROM accounts WHERE platform = 'grok-video');

UPDATE openai_video_characters
SET platform = 'grok-video',
    updated_at = NOW()
WHERE platform = 'grok'
  AND account_id IN (SELECT id FROM accounts WHERE platform = 'grok-video');
