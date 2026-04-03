-- Comment Queries

-- name: CreateComment :one
INSERT INTO post.comments (post_id, author_id, parent_id, content)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCommentByID :one
SELECT * FROM post.comments
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCommentsByPost :many
SELECT * FROM post.comments
WHERE post_id = sqlc.arg('post_id')
  AND parent_id IS NULL
  AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetReplies :many
SELECT * FROM post.comments
WHERE parent_id = sqlc.arg('parent_id')
  AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCommentsByPost :one
SELECT COUNT(*) FROM post.comments
WHERE post_id = $1 AND parent_id IS NULL AND deleted_at IS NULL;

-- name: DeleteComment :exec
UPDATE post.comments
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetReplyCountsBatch :many
SELECT parent_id AS root_id, COUNT(*) AS count
FROM post.comments
WHERE parent_id = ANY($1::uuid[])
  AND deleted_at IS NULL
GROUP BY parent_id;

-- name: GetRepliesPreview :many
SELECT id, post_id, author_id, parent_id, content, created_at, updated_at, deleted_at, like_count
FROM (
    SELECT id, post_id, author_id, parent_id, content, created_at, updated_at, deleted_at, like_count,
           ROW_NUMBER() OVER (PARTITION BY parent_id ORDER BY created_at ASC) AS rn
    FROM post.comments
    WHERE parent_id = ANY($1::uuid[])
      AND deleted_at IS NULL
) ranked
WHERE rn <= $2::int
ORDER BY parent_id, created_at ASC;
