-- User Queries

-- name: CreateUser :one
INSERT INTO usr.users (
    username,
    email,
    password_hash,
    display_name,
    created_by
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM usr.users
WHERE id = $1 AND is_active = true;

-- name: GetUserByIDAny :one
SELECT * FROM usr.users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM usr.users
WHERE username = $1 AND is_active = true;

-- name: GetUserByEmail :one
SELECT * FROM usr.users
WHERE email = $1 AND is_active = true;

-- name: ExistsUsername :one
SELECT EXISTS(
    SELECT 1 FROM usr.users
    WHERE username = $1
);

-- name: ExistsEmail :one
SELECT EXISTS(
    SELECT 1 FROM usr.users
    WHERE email = $1
);

-- name: ExistsEmailExcludingUser :one
SELECT EXISTS(
    SELECT 1 FROM usr.users
    WHERE email = $1 AND id != $2
);

-- name: UpdateUser :one
UPDATE usr.users
SET
    email      = COALESCE(sqlc.narg('email'), email),
    updated_at = NOW(),
    updated_by = sqlc.narg('updated_by')
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE usr.users
SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    bio          = COALESCE(sqlc.narg('bio'), bio),
    avatar_key   = COALESCE(sqlc.narg('avatar_key'), avatar_key),
    cover_key    = COALESCE(sqlc.narg('cover_key'), cover_key),
    website      = COALESCE(sqlc.narg('website'), website),
    location     = COALESCE(sqlc.narg('location'), location),
    updated_at   = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE usr.users
SET
    password_hash = $2,
    updated_at    = NOW(),
    updated_by    = $3
WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE usr.users
SET
    is_active  = false,
    updated_at = NOW(),
    updated_by = $2
WHERE id = $1;

-- name: SearchUsers :many
SELECT * FROM usr.users
WHERE is_active = true
  AND (
    sqlc.narg('query')::text IS NULL
    OR sqlc.narg('query')::text = ''
    OR username ILIKE '%' || sqlc.narg('query')::text || '%'
    OR email    ILIKE '%' || sqlc.narg('query')::text || '%'
  )
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetUsersByIDs :many
SELECT * FROM usr.users
WHERE id = ANY($1::uuid[])
  AND is_active = true;

-- name: GetUsersByIDsAny :many
SELECT * FROM usr.users
WHERE id = ANY($1::uuid[]);

-- name: GetUsersByUsernames :many
SELECT * FROM usr.users
WHERE username = ANY($1::text[])
  AND is_active = true;

-- name: CountSearchUsers :one
SELECT COUNT(*) FROM usr.users
WHERE is_active = true
  AND (
    sqlc.narg('query')::text IS NULL
    OR sqlc.narg('query')::text = ''
    OR username ILIKE '%' || sqlc.narg('query')::text || '%'
    OR email    ILIKE '%' || sqlc.narg('query')::text || '%'
  );

-- name: SearchUsersByQuery :many
SELECT * FROM usr.users
WHERE is_active = true
  AND (
    username     ILIKE '%' || sqlc.arg('query')::text || '%'
    OR display_name ILIKE '%' || sqlc.arg('query')::text || '%'
  )
ORDER BY
  CASE WHEN username ILIKE sqlc.arg('query')::text || '%' THEN 0 ELSE 1 END,
  follower_count DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- Admin Queries

-- name: AdminListUsers :many
SELECT * FROM usr.users
WHERE (
    sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active')::boolean
  )
  AND (
    sqlc.narg('query')::text IS NULL
    OR sqlc.narg('query')::text = ''
    OR username     ILIKE '%' || sqlc.narg('query')::text || '%'
    OR email        ILIKE '%' || sqlc.narg('query')::text || '%'
    OR display_name ILIKE '%' || sqlc.narg('query')::text || '%'
  )
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: AdminCountUsers :one
SELECT COUNT(*) FROM usr.users
WHERE (
    sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active')::boolean
  )
  AND (
    sqlc.narg('query')::text IS NULL
    OR sqlc.narg('query')::text = ''
    OR username     ILIKE '%' || sqlc.narg('query')::text || '%'
    OR email        ILIKE '%' || sqlc.narg('query')::text || '%'
    OR display_name ILIKE '%' || sqlc.narg('query')::text || '%'
  );

-- name: AdminSetUserActive :exec
UPDATE usr.users
SET is_active  = $2,
    updated_at = NOW(),
    updated_by = $3
WHERE id = $1;

-- name: ListAllActiveUserIDs :many
SELECT id FROM usr.users WHERE is_active = true ORDER BY created_at ASC;
