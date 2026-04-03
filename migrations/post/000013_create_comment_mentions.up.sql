CREATE TABLE IF NOT EXISTS post.comment_mentions (
    comment_id UUID        NOT NULL REFERENCES post.comments(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (comment_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_comment_mentions_comment_id ON post.comment_mentions(comment_id);
CREATE INDEX IF NOT EXISTS idx_comment_mentions_user_id    ON post.comment_mentions(user_id);
