INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES
    ('grok', 'Grok', 'openai_compatible', '', '["apikey"]'::jsonb, '["videos"]'::jsonb, '#111827', TRUE, TRUE),
    ('kling_audio', '可灵 Audio', 'openai_compatible', '', '["apikey"]'::jsonb, '["audio"]'::jsonb, '#0EA5E9', TRUE, TRUE)
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    protocol = EXCLUDED.protocol,
    base_url = CASE
        WHEN platforms.base_url = 'https://api.mengfactory.cn' THEN EXCLUDED.base_url
        ELSE platforms.base_url
    END,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    color = EXCLUDED.color,
    enabled = TRUE,
    builtin = TRUE,
    updated_at = NOW();
