INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES
    ('openai-audio', 'OpenAI Audio', 'openai_compatible', '', '["apikey"]'::jsonb, '["chat_completions","audio"]'::jsonb, '#0F766E', TRUE, TRUE)
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    protocol = EXCLUDED.protocol,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    color = EXCLUDED.color,
    enabled = TRUE,
    builtin = TRUE,
    updated_at = NOW();
