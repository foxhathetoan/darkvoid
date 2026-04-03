-- Post Queries

-- name: CreatePost :one
INSERT INTO post.posts (author_id, content, visibility)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPostByID :one
SELECT * FROM post.posts
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPostsByAuthor :many
SELECT * FROM post.posts
WHERE author_id = sqlc.arg('author_id') AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPostsByAuthor :one
SELECT COUNT(*) FROM post.posts
WHERE author_id = $1 AND deleted_at IS NULL;

-- name: UpdatePost :one
UPDATE post.posts
SET content = $2, visibility = $3, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeletePost :exec
UPDATE post.posts
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetFeedPosts :many
SELECT * FROM post.posts
WHERE author_id = ANY(sqlc.arg('author_ids')::uuid[])
  AND deleted_at IS NULL
  AND visibility IN ('public', 'followers')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountFeedPosts :one
SELECT COUNT(*) FROM post.posts
WHERE author_id = ANY(sqlc.arg('author_ids')::uuid[])
  AND deleted_at IS NULL
  AND visibility IN ('public', 'followers');

-- name: GetDiscoverPosts :many
SELECT * FROM post.posts
WHERE deleted_at IS NULL
  AND visibility = 'public'
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountDiscoverPosts :one
SELECT COUNT(*) FROM post.posts
WHERE deleted_at IS NULL
  AND visibility = 'public';

-- name: GetDiscoverWithCursor :many
SELECT * FROM post.posts
WHERE deleted_at IS NULL
  AND visibility = 'public'
  AND (created_at, id) < (sqlc.arg('cursor_created_at')::timestamptz, sqlc.arg('cursor_post_id')::uuid)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: GetFollowingPosts :many
SELECT * FROM post.posts
WHERE author_id = ANY(sqlc.arg('author_ids')::uuid[])
  AND created_at > NOW() - INTERVAL '7 days'
  AND visibility IN ('public', 'followers')
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT sqlc.arg('limit');

-- name: GetTrendingPosts :many
SELECT * FROM post.posts
WHERE created_at > NOW() - INTERVAL '24 hours'
  AND visibility = 'public'
  AND deleted_at IS NULL
ORDER BY like_count DESC
LIMIT sqlc.arg('limit');

-- name: GetUserPostsWithCursor :many
SELECT * FROM post.posts
WHERE author_id = $1
  AND deleted_at IS NULL
  AND ($4::text = '' OR visibility = $4::text)
  AND (created_at, id) < ($2::timestamptz, $3::uuid)
ORDER BY created_at DESC, id DESC
LIMIT $5;

-- name: GetFollowingPostsWithCursor :many
SELECT id, author_id, content, visibility, created_at, updated_at, deleted_at, like_count, comment_count
FROM post.posts
WHERE author_id = ANY($1::uuid[])
  AND visibility = 'public'
  AND deleted_at IS NULL
  AND (created_at < $2::timestamptz
       OR (created_at = $2::timestamptz AND id < $3::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $4;
