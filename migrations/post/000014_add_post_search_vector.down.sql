DROP INDEX IF EXISTS post.idx_posts_search_vector;
ALTER TABLE post.posts DROP COLUMN IF EXISTS search_vector;
