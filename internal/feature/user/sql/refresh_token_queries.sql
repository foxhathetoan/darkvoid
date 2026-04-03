-- name: CreateRefreshToken :one
INSERT INTO usr.refresh_tokens (
    token,
    user_id,
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetRefreshTokenByToken :one
SELECT * FROM usr.refresh_tokens
WHERE token = $1 AND is_revoked = false
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE usr.refresh_tokens
SET is_revoked = true, revoked_at = NOW()
WHERE token = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE usr.refresh_tokens
SET is_revoked = true, revoked_at = NOW()
WHERE user_id = $1 AND is_revoked = false;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM usr.refresh_tokens
WHERE expires_at < NOW();

-- name: GetActiveRefreshTokensByUserID :many
SELECT * FROM usr.refresh_tokens
WHERE user_id = $1 AND is_revoked = false AND expires_at > NOW()
ORDER BY created_at DESC;
