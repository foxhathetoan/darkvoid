-- Like Queries

-- name: LikePost :exec
INSERT INTO post.likes (user_id, post_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UnlikePost :exec
DELETE FROM post.likes
WHERE user_id = $1 AND post_id = $2;

-- name: IsLiked :one
SELECT EXISTS(
    SELECT 1 FROM post.likes
    WHERE user_id = $1 AND post_id = $2
);

-- name: GetLikedPostIDs :many
SELECT post_id FROM post.likes
WHERE user_id = $1
  AND post_id = ANY($2::uuid[]);

-- name: CountLikes :one
SELECT COUNT(*) FROM post.likes
WHERE post_id = $1;
