CREATE TABLE post.hashtags (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE post.post_hashtags (
    post_id    UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    hashtag_id UUID        NOT NULL REFERENCES post.hashtags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, hashtag_id)
);

-- For trending query: COUNT per hashtag in a time window
CREATE INDEX idx_post_hashtags_hashtag_created
    ON post.post_hashtags (hashtag_id, created_at DESC);
