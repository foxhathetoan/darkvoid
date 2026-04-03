CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_users_username_trgm
    ON usr.users USING gin(username gin_trgm_ops)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_users_display_name_trgm
    ON usr.users USING gin(display_name gin_trgm_ops)
    WHERE is_active = true;
