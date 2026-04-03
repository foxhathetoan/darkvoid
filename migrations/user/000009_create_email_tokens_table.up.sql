CREATE TABLE usr.email_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES usr.users(id) ON DELETE CASCADE,
    token      VARCHAR(255) NOT NULL UNIQUE,
    type       VARCHAR(30)  NOT NULL, -- 'verify_email' | 'reset_password'
    expires_at TIMESTAMP    NOT NULL,
    used_at    TIMESTAMP,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_usr_email_tokens_token   ON usr.email_tokens(token);
CREATE INDEX idx_usr_email_tokens_user_id ON usr.email_tokens(user_id, type);
