ALTER TABLE post.posts ADD COLUMN like_count BIGINT NOT NULL DEFAULT 0;

-- Backfill existing counts
UPDATE post.posts p
SET like_count = (
    SELECT COUNT(*) FROM post.likes l WHERE l.post_id = p.id
);

-- Trigger to keep like_count in sync
CREATE OR REPLACE FUNCTION post.update_like_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE post.posts SET like_count = like_count + 1 WHERE id = NEW.post_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE post.posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = OLD.post_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_like_count
AFTER INSERT OR DELETE ON post.likes
FOR EACH ROW EXECUTE FUNCTION post.update_like_count();

CREATE INDEX idx_posts_like_count ON post.posts(like_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_trending ON post.posts(created_at DESC, like_count DESC) WHERE deleted_at IS NULL AND visibility = 'public';
