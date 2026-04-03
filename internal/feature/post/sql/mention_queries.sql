-- name: InsertMention :exec
INSERT INTO post.post_mentions (post_id, user_id)
VALUES ($1, $2)
ON CONFLICT (post_id, user_id) DO NOTHING;

-- name: DeleteMentionsByPost :exec
DELETE FROM post.post_mentions WHERE post_id = $1;

-- name: GetMentionsByPost :many
SELECT post_id, user_id, created_at FROM post.post_mentions WHERE post_id = $1;

-- name: GetMentionsBatch :many
SELECT post_id, user_id, created_at FROM post.post_mentions WHERE post_id = ANY($1::uuid[]);
