CREATE TABLE post.comment_likes (
    user_id    UUID        NOT NULL,
    comment_id UUID        NOT NULL REFERENCES post.comments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, comment_id)
);

CREATE INDEX idx_comment_likes_comment_id ON post.comment_likes(comment_id);

ALTER TABLE post.comments ADD COLUMN like_count BIGINT NOT NULL DEFAULT 0;

-- Trigger to keep like_count in sync
CREATE OR REPLACE FUNCTION post.update_comment_like_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE post.comments SET like_count = like_count + 1 WHERE id = NEW.comment_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE post.comments SET like_count = GREATEST(like_count - 1, 0) WHERE id = OLD.comment_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_comment_like_count
AFTER INSERT OR DELETE ON post.comment_likes
FOR EACH ROW EXECUTE FUNCTION post.update_comment_like_count();
