-- User-Role Relationship Queries

-- name: AssignRoleToUser :exec
INSERT INTO usr.user_roles (
    user_id,
    role_id,
    assigned_by
) VALUES (
    $1, $2, $3
);

-- name: RemoveRoleFromUser :exec
DELETE FROM usr.user_roles
WHERE user_id = $1 AND role_id = $2;

-- name: GetUserRoles :many
SELECT r.* FROM usr.roles r
INNER JOIN usr.user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = $1
ORDER BY r.name;

-- name: CheckUserHasRole :one
SELECT EXISTS(
    SELECT 1 FROM usr.user_roles
    WHERE user_id = $1 AND role_id = $2
);
