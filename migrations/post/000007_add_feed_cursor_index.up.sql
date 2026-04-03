-- Composite partial index for following feed cursor pagination.
-- Supports bitmap index scan on author_id = ANY($1::uuid[]) + row value
-- comparison (created_at, id) < (cursor_ts, cursor_id).
-- Partial (WHERE deleted_at IS NULL) keeps the index small by excluding soft-deleted rows.
CREATE INDEX idx_posts_author_created_id
    ON post.posts (author_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
