-- Follow Queries

-- name: Follow :exec
INSERT INTO usr.follows (follower_id, followee_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: Unfollow :exec
DELETE FROM usr.follows
WHERE follower_id = $1 AND followee_id = $2;

-- name: IsFollowing :one
SELECT EXISTS(
    SELECT 1 FROM usr.follows
    WHERE follower_id = $1 AND followee_id = $2
);

-- name: GetFollowers :many
SELECT follower_id, followee_id, created_at
FROM usr.follows
WHERE followee_id = sqlc.arg('followee_id')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetFollowing :many
SELECT follower_id, followee_id, created_at
FROM usr.follows
WHERE follower_id = sqlc.arg('follower_id')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountFollowers :one
SELECT COUNT(*) FROM usr.follows
WHERE followee_id = $1;

-- name: CountFollowing :one
SELECT COUNT(*) FROM usr.follows
WHERE follower_id = $1;
