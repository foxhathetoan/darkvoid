CREATE TABLE post.likes (
    user_id    UUID        NOT NULL,
    post_id    UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX idx_likes_post_id ON post.likes(post_id);
