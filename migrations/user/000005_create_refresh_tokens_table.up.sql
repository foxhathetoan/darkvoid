CREATE TABLE usr.refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    token      VARCHAR(255) UNIQUE NOT NULL,
    user_id    UUID         NOT NULL REFERENCES usr.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP    NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP,
    is_revoked BOOLEAN      NOT NULL DEFAULT false
);

CREATE INDEX idx_usr_refresh_tokens_token      ON usr.refresh_tokens(token);
CREATE INDEX idx_usr_refresh_tokens_user_id    ON usr.refresh_tokens(user_id);
CREATE INDEX idx_usr_refresh_tokens_expires_at ON usr.refresh_tokens(expires_at);
CREATE INDEX idx_usr_refresh_tokens_is_revoked ON usr.refresh_tokens(is_revoked);
