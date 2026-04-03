DROP TRIGGER IF EXISTS trg_comment_like_count ON post.comment_likes;
DROP FUNCTION IF EXISTS post.update_comment_like_count();
ALTER TABLE post.comments DROP COLUMN IF EXISTS like_count;
DROP TABLE IF EXISTS post.comment_likes;
