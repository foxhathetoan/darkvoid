CREATE TABLE post.comments (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    post_id    UUID        NOT NULL REFERENCES post.posts(id) ON DELETE CASCADE,
    author_id  UUID        NOT NULL,
    parent_id  UUID        REFERENCES post.comments(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (id)
);

CREATE INDEX idx_comments_post_id ON post.comments(post_id, created_at ASC) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_parent_id ON post.comments(parent_id) WHERE parent_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_comments_author_id ON post.comments(author_id);
