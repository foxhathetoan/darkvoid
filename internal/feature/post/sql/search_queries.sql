-- name: SearchPosts :many
SELECT id, author_id, content, visibility, created_at, updated_at, deleted_at, like_count, comment_count
FROM post.posts
WHERE deleted_at IS NULL
  AND visibility = 'public'
  AND search_vector @@ plainto_tsquery('english', $1)
ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC, created_at DESC
LIMIT $2 OFFSET $3;
