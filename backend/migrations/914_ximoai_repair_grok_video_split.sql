CREATE TEMP TABLE ximoai_legacy_grok_platform ON COMMIT DROP AS
SELECT base_url
FROM (
    SELECT NULLIF(BTRIM(base_url), '') AS base_url
    FROM platforms
    WHERE slug = 'grok'
) AS p
WHERE p.base_url IS NOT NULL
  AND LOWER(TRIM(TRAILING '/' FROM p.base_url)) NOT IN (
      'https://api.x.ai',
      'https://api.x.ai/v1',
      'https://cli-chat-proxy.grok.com',
      'https://cli-chat-proxy.grok.com/v1'
  )
LIMIT 1;

INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES (
    'grok-video',
    'Grok-video',
    'openai_compatible',
    COALESCE((SELECT base_url FROM ximoai_legacy_grok_platform LIMIT 1), ''),
    '["apikey"]'::jsonb,
    '["videos"]'::jsonb,
    '#111827',
    TRUE,
    TRUE
)
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    base_url = CASE
        WHEN NULLIF(BTRIM(platforms.base_url), '') IS NULL THEN EXCLUDED.base_url
        ELSE platforms.base_url
    END,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    color = EXCLUDED.color,
    enabled = TRUE,
    builtin = TRUE,
    updated_at = NOW();

CREATE TEMP TABLE ximoai_grok_video_account_ids (
    id BIGINT PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO ximoai_grok_video_account_ids (id)
SELECT a.id
FROM accounts AS a
WHERE a.platform = 'grok'
  AND a.type IN ('apikey', 'api_key')
  AND NOT (COALESCE(a.credentials, '{}'::jsonb) ? 'access_token')
  AND NOT (COALESCE(a.credentials, '{}'::jsonb) ? 'refresh_token')
  AND (
      (
          NULLIF(BTRIM(COALESCE(a.credentials ->> 'base_url', '')), '') IS NULL
          AND EXISTS (SELECT 1 FROM ximoai_legacy_grok_platform)
      )
      OR (
          NULLIF(BTRIM(COALESCE(a.credentials ->> 'base_url', '')), '') IS NOT NULL
          AND LOWER(TRIM(TRAILING '/' FROM COALESCE(a.credentials ->> 'base_url', ''))) NOT IN (
              'https://api.x.ai',
              'https://api.x.ai/v1',
              'https://cli-chat-proxy.grok.com',
              'https://cli-chat-proxy.grok.com/v1'
          )
      )
      OR COALESCE(a.credentials, '{}'::jsonb) ? 'platform_protocol'
      OR EXISTS (
          SELECT 1
          FROM openai_video_jobs AS vj
          WHERE vj.account_id = a.id
            AND vj.platform = 'grok'
      )
      OR EXISTS (
          SELECT 1
          FROM openai_video_characters AS vc
          WHERE vc.account_id = a.id
            AND vc.platform = 'grok'
      )
  )
ON CONFLICT DO NOTHING;

UPDATE accounts AS a
SET platform = 'grok-video',
    credentials = jsonb_set(
        CASE
            WHEN legacy.base_url IS NOT NULL
              AND NULLIF(BTRIM(COALESCE(a.credentials ->> 'base_url', '')), '') IS NULL
            THEN jsonb_set(
                COALESCE(a.credentials, '{}'::jsonb),
                '{base_url}',
                to_jsonb(legacy.base_url),
                true
            )
            ELSE COALESCE(a.credentials, '{}'::jsonb)
        END,
        '{platform_protocol}',
        to_jsonb(COALESCE(NULLIF(BTRIM(a.credentials ->> 'platform_protocol'), ''), 'openai_compatible')::text),
        true
    ),
    updated_at = NOW()
FROM ximoai_grok_video_account_ids AS ids
LEFT JOIN LATERAL (
    SELECT base_url
    FROM ximoai_legacy_grok_platform
    LIMIT 1
) AS legacy ON TRUE
WHERE a.id = ids.id;

UPDATE groups AS g
SET platform = 'grok-video',
    updated_at = NOW()
WHERE g.platform = 'grok'
  AND g.require_oauth_only = FALSE
  AND EXISTS (
      SELECT 1
      FROM account_groups AS ag
      JOIN ximoai_grok_video_account_ids AS ids ON ids.id = ag.account_id
      WHERE ag.group_id = g.id
  )
  AND NOT EXISTS (
      SELECT 1
      FROM account_groups AS ag
      JOIN accounts AS a ON a.id = ag.account_id
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
  AND NOT (model_mapping ? 'grok-video')
  AND EXISTS (SELECT 1 FROM ximoai_grok_video_account_ids);

UPDATE openai_video_jobs
SET platform = 'grok-video',
    updated_at = NOW()
WHERE platform = 'grok'
  AND account_id IN (SELECT id FROM ximoai_grok_video_account_ids);

UPDATE openai_video_characters
SET platform = 'grok-video',
    updated_at = NOW()
WHERE platform = 'grok'
  AND account_id IN (SELECT id FROM ximoai_grok_video_account_ids);

INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES (
    'grok',
    'Grok',
    'openai_compatible',
    'https://api.x.ai/v1',
    '["oauth","apikey"]'::jsonb,
    '["responses","chat_completions","videos"]'::jsonb,
    '#111827',
    TRUE,
    TRUE
)
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    protocol = EXCLUDED.protocol,
    base_url = EXCLUDED.base_url,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    color = EXCLUDED.color,
    enabled = TRUE,
    builtin = TRUE,
    updated_at = NOW();
