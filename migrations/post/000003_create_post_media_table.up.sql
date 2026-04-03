CREATE TABLE post.post_media (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    post_id    UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    media_key  TEXT        NOT NULL,
    media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('image', 'video')),
    position   INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE INDEX idx_post_media_post_id ON post.post_media(post_id, position);
