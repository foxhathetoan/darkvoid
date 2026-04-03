ALTER TABLE post.posts
    ADD COLUMN IF NOT EXISTS search_vector tsvector
        GENERATED ALWAYS AS (to_tsvector('english', coalesce(content, ''))) STORED;

CREATE INDEX IF NOT EXISTS idx_posts_search_vector
    ON post.posts USING gin(search_vector)
    WHERE deleted_at IS NULL AND visibility = 'public';
