-- name: AddCommentMedia :one
INSERT INTO post.comment_media (comment_id, media_key, media_type, position)
VALUES ($1, $2, $3, $4)
RETURNING id, comment_id, media_key, media_type, position, created_at;

-- name: GetCommentMedia :many
SELECT id, comment_id, media_key, media_type, position, created_at FROM post.comment_media
WHERE comment_id = $1
ORDER BY position ASC;

-- name: GetCommentMediaBatch :many
SELECT id, comment_id, media_key, media_type, position, created_at FROM post.comment_media
WHERE comment_id = ANY($1::uuid[])
ORDER BY comment_id, position ASC;

-- name: DeleteAllCommentMedia :exec
DELETE FROM post.comment_media
WHERE comment_id = $1;
