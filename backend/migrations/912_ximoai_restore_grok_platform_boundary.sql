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
