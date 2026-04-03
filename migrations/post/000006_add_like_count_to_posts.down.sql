DROP TRIGGER IF EXISTS trg_like_count ON post.likes;
DROP FUNCTION IF EXISTS post.update_like_count();
DROP INDEX IF EXISTS post.idx_posts_trending;
DROP INDEX IF EXISTS post.idx_posts_like_count;
ALTER TABLE post.posts DROP COLUMN IF EXISTS like_count;
