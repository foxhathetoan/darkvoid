-- name: CreateNotification :one
INSERT INTO notification.notifications (recipient_id, actor_id, type, target_id, secondary_id, group_key)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (recipient_id, actor_id, type, group_key)
DO UPDATE SET created_at = NOW(), is_read = FALSE
RETURNING id, recipient_id, actor_id, type, target_id, secondary_id, group_key, is_read, created_at;

-- name: GetNotificationsWithCursor :many
SELECT id, recipient_id, actor_id, type, target_id, secondary_id, group_key, is_read, created_at
FROM notification.notifications
WHERE recipient_id = $1
  AND (
      created_at < $2::timestamptz
      OR (created_at = $2::timestamptz AND id < $3::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT $4;

-- name: MarkAsRead :exec
UPDATE notification.notifications
SET is_read = TRUE
WHERE id = $1 AND recipient_id = $2 AND is_read = FALSE;

-- name: MarkAllAsRead :exec
UPDATE notification.notifications
SET is_read = TRUE
WHERE recipient_id = $1 AND is_read = FALSE;

-- name: CountUnread :one
SELECT COUNT(*)
FROM notification.notifications
WHERE recipient_id = $1 AND is_read = FALSE;

-- name: GetGroupActors :many
SELECT actor_id, COUNT(*) OVER() AS total_count
FROM notification.notifications
WHERE recipient_id = $1 AND group_key = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: DeleteByActorAndGroupKey :exec
DELETE FROM notification.notifications
WHERE actor_id = $1 AND group_key = $2;

-- name: CreateSystemNotification :one
INSERT INTO notification.notifications (recipient_id, actor_id, type, group_key, message)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
