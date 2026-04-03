CREATE TABLE notification.notifications (
    id            UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    recipient_id  UUID        NOT NULL,
    actor_id      UUID        NOT NULL,
    type          TEXT        NOT NULL,
    target_id     UUID,
    secondary_id  UUID,
    group_key     TEXT        NOT NULL,
    is_read       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cursor-based pagination: newest first, keyset on (created_at, id)
CREATE INDEX idx_notif_recipient_cursor
    ON notification.notifications (recipient_id, created_at DESC, id DESC);

-- Unread count: partial index for fast COUNT
CREATE INDEX idx_notif_unread
    ON notification.notifications (recipient_id) WHERE is_read = FALSE;

-- Dedup: same actor + type + group_key → upsert instead of duplicate
CREATE UNIQUE INDEX idx_notif_dedup
    ON notification.notifications (recipient_id, actor_id, type, group_key);
