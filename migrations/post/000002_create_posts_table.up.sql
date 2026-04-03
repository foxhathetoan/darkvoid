CREATE TABLE post.posts (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    author_id  UUID        NOT NULL,
    content    TEXT        NOT NULL DEFAULT '',
    visibility VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'followers', 'private')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (id)
);

CREATE INDEX idx_posts_author_id ON post.posts(author_id);
CREATE INDEX idx_posts_created_at ON post.posts(created_at DESC);
CREATE INDEX idx_posts_author_created ON post.posts(author_id, created_at DESC) WHERE deleted_at IS NULL;
