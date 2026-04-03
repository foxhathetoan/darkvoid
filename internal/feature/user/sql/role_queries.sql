-- Role Queries

-- name: CreateRole :one
INSERT INTO usr.roles (
    name,
    description
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM usr.roles
WHERE id = $1;

-- name: GetRoleByName :one
SELECT * FROM usr.roles
WHERE name = $1;

-- name: ListRoles :many
SELECT * FROM usr.roles
ORDER BY name;
