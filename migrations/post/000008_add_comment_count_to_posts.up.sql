ALTER TABLE post.posts ADD COLUMN comment_count BIGINT NOT NULL DEFAULT 0;

-- Backfill existing counts (all comments including replies)
UPDATE post.posts p
SET comment_count = (
    SELECT COUNT(*) FROM post.comments c WHERE c.post_id = p.id AND c.deleted_at IS NULL
);

-- Trigger to keep comment_count in sync (INSERT and soft-delete UPDATE)
CREATE OR REPLACE FUNCTION post.update_comment_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE post.posts SET comment_count = comment_count + 1 WHERE id = NEW.post_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        UPDATE post.posts SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = NEW.post_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_comment_count
AFTER INSERT OR UPDATE ON post.comments
FOR EACH ROW EXECUTE FUNCTION post.update_comment_count();
