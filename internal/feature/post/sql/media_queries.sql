-- Post Media Queries

-- name: AddPostMedia :one
INSERT INTO post.post_media (post_id, media_key, media_type, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPostMedia :many
SELECT * FROM post.post_media
WHERE post_id = $1
ORDER BY position ASC;

-- name: DeletePostMedia :exec
DELETE FROM post.post_media
WHERE id = $1 AND post_id = $2;

-- name: DeleteAllPostMedia :exec
DELETE FROM post.post_media
WHERE post_id = $1;
