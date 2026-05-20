CREATE TABLE IF NOT EXISTS openai_video_characters (
    id              BIGSERIAL PRIMARY KEY,
    character_id    VARCHAR(255) NOT NULL UNIQUE,
    platform        VARCHAR(64) NOT NULL,
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    group_id        BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    api_key_id      BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    user_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    response_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_openai_video_characters_platform ON openai_video_characters(platform);
CREATE INDEX IF NOT EXISTS idx_openai_video_characters_account ON openai_video_characters(account_id);
CREATE INDEX IF NOT EXISTS idx_openai_video_characters_group ON openai_video_characters(group_id);
CREATE INDEX IF NOT EXISTS idx_openai_video_characters_api_key ON openai_video_characters(api_key_id);
CREATE INDEX IF NOT EXISTS idx_openai_video_characters_user ON openai_video_characters(user_id);
