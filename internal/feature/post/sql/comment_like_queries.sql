-- name: LikeComment :exec
INSERT INTO post.comment_likes (user_id, comment_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UnlikeComment :exec
DELETE FROM post.comment_likes
WHERE user_id = $1 AND comment_id = $2;

-- name: IsCommentLiked :one
SELECT EXISTS(
    SELECT 1 FROM post.comment_likes
    WHERE user_id = $1 AND comment_id = $2
);

-- name: GetLikedCommentIDs :many
SELECT comment_id FROM post.comment_likes
WHERE user_id = $1
  AND comment_id = ANY($2::uuid[]);
