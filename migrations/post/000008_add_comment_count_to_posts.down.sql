DROP TRIGGER IF EXISTS trg_comment_count ON post.comments;
DROP FUNCTION IF EXISTS post.update_comment_count();
ALTER TABLE post.posts DROP COLUMN IF EXISTS comment_count;
