CREATE TABLE IF NOT EXISTS post.post_mentions (
    post_id    UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_post_mentions_post_id ON post.post_mentions(post_id);
CREATE INDEX IF NOT EXISTS idx_post_mentions_user_id ON post.post_mentions(user_id);
