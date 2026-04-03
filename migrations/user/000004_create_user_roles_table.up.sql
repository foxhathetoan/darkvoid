CREATE TABLE usr.user_roles (
    user_id     UUID      NOT NULL REFERENCES usr.users(id) ON DELETE CASCADE,
    role_id     UUID      NOT NULL REFERENCES usr.roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    assigned_by UUID,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_usr_user_roles_user_id ON usr.user_roles(user_id);
CREATE INDEX idx_usr_user_roles_role_id ON usr.user_roles(role_id);
