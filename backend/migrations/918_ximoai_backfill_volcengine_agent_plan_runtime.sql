UPDATE accounts AS account
SET credentials = jsonb_set(
        COALESCE(account.credentials, '{}'::jsonb),
        '{platform_kind}',
        to_jsonb('volcengine_agent_plan'::text),
        TRUE
    ),
    updated_at = NOW()
FROM platforms AS platform
WHERE account.platform = platform.slug
  AND (
        platform.kind = 'volcengine_agent_plan'
        OR (
            platform.protocol = 'native'
            AND RTRIM(platform.base_url, '/') = 'https://ark.cn-beijing.volces.com/api/plan/v3'
        )
    )
  AND COALESCE(account.credentials->>'platform_kind', '') <> 'volcengine_agent_plan';
