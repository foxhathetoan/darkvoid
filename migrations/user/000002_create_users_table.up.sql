CREATE TABLE usr.users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50)  UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    display_name  VARCHAR(100) NOT NULL DEFAULT '',
    bio           TEXT,
    avatar_key    VARCHAR(255),
    cover_key     VARCHAR(255),
    website       VARCHAR(255),
    location      VARCHAR(100),
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP,
    created_by    UUID,
    updated_by    UUID
);

CREATE INDEX idx_usr_users_username   ON usr.users(username);
CREATE INDEX idx_usr_users_email      ON usr.users(email);
CREATE INDEX idx_usr_users_is_active  ON usr.users(is_active);
CREATE INDEX idx_usr_users_created_at ON usr.users(created_at DESC);
