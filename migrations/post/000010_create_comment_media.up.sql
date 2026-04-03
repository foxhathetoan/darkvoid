CREATE TABLE post.comment_media (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    comment_id UUID        NOT NULL REFERENCES post.comments(id) ON DELETE CASCADE,
    media_key  TEXT        NOT NULL,
    media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('image', 'video')),
    position   INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE INDEX idx_comment_media_comment_id ON post.comment_media(comment_id, position);
