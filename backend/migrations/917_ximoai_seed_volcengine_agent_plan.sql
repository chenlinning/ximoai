INSERT INTO platforms (slug, kind, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES (
    'volcengine-agent-plan',
    'volcengine_agent_plan',
    'Volcengine Agent Plan',
    'native',
    'https://ark.cn-beijing.volces.com/api/plan/v3',
    '["apikey"]'::jsonb,
    '["images","audio"]'::jsonb,
    '#E5484D',
    TRUE,
    TRUE
)
ON CONFLICT (slug) DO UPDATE SET
    kind = EXCLUDED.kind,
    display_name = EXCLUDED.display_name,
    protocol = EXCLUDED.protocol,
    base_url = EXCLUDED.base_url,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    builtin = TRUE,
    updated_at = NOW();
