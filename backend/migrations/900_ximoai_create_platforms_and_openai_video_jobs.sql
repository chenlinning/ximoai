CREATE TABLE IF NOT EXISTS platforms (
    slug         VARCHAR(64) PRIMARY KEY,
    display_name VARCHAR(100) NOT NULL,
    protocol     VARCHAR(32) NOT NULL DEFAULT 'native',
    base_url     TEXT NOT NULL DEFAULT '',
    auth_modes   JSONB NOT NULL DEFAULT '[]'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    color        VARCHAR(32) NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    builtin      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platforms_enabled ON platforms(enabled);
CREATE INDEX IF NOT EXISTS idx_platforms_protocol ON platforms(protocol);

INSERT INTO platforms (slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled, builtin)
VALUES
    ('anthropic', 'Anthropic', 'native', '', '["oauth","setup-token","apikey","bedrock"]'::jsonb, '["messages"]'::jsonb, '#D97706', TRUE, TRUE),
    ('openai', 'OpenAI', 'openai', 'https://api.openai.com', '["oauth","apikey"]'::jsonb, '["responses","chat_completions","images","videos","codex"]'::jsonb, '#10A37F', TRUE, TRUE),
    ('gemini', 'Gemini', 'native', '', '["oauth","apikey","service_account"]'::jsonb, '["messages","native_gemini"]'::jsonb, '#4285F4', TRUE, TRUE),
    ('antigravity', 'Antigravity', 'native', '', '["oauth","upstream","apikey"]'::jsonb, '["messages","native_gemini"]'::jsonb, '#7C3AED', TRUE, TRUE)
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    protocol = EXCLUDED.protocol,
    auth_modes = EXCLUDED.auth_modes,
    capabilities = EXCLUDED.capabilities,
    color = CASE WHEN platforms.color = '' THEN EXCLUDED.color ELSE platforms.color END,
    builtin = TRUE,
    updated_at = NOW();

CREATE TABLE IF NOT EXISTS openai_video_jobs (
    id               BIGSERIAL PRIMARY KEY,
    video_id         VARCHAR(255) NOT NULL UNIQUE,
    platform         VARCHAR(64) NOT NULL,
    account_id       BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    group_id         BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    api_key_id       BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    user_id          BIGINT REFERENCES users(id) ON DELETE SET NULL,
    channel_id       BIGINT REFERENCES channels(id) ON DELETE SET NULL,
    request_model    VARCHAR(255) NOT NULL DEFAULT '',
    upstream_model   VARCHAR(255) NOT NULL DEFAULT '',
    status           VARCHAR(64) NOT NULL DEFAULT '',
    response_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_openai_video_jobs_platform ON openai_video_jobs(platform);
CREATE INDEX IF NOT EXISTS idx_openai_video_jobs_account ON openai_video_jobs(account_id);
CREATE INDEX IF NOT EXISTS idx_openai_video_jobs_group ON openai_video_jobs(group_id);
CREATE INDEX IF NOT EXISTS idx_openai_video_jobs_api_key ON openai_video_jobs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_openai_video_jobs_created_at ON openai_video_jobs(created_at DESC);
