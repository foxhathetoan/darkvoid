package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

// RoleRepository handles all DB operations for roles and user-role assignments.
type RoleRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewRoleRepository creates a RoleRepository backed by the given connection pool.
func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// ListRoles returns all roles ordered by name.
func (r *RoleRepository) ListRoles(ctx context.Context) ([]*entity.Role, error) {
	rows, err := r.queries.ListRoles(ctx)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	roles := make([]*entity.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, dbRoleToEntity(row))
	}
	return roles, nil
}

// CreateRole creates a new role and returns the persisted entity.
func (r *RoleRepository) CreateRole(ctx context.Context, name string, description *string) (*entity.Role, error) {
	row, err := r.queries.CreateRole(ctx, db.CreateRoleParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbRoleToEntity(row), nil
}

// GetRoleByID fetches a role by primary key.
func (r *RoleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	row, err := r.queries.GetRoleByID(ctx, id)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbRoleToEntity(row), nil
}

// GetRoleByName fetches a role by its unique name.
func (r *RoleRepository) GetRoleByName(ctx context.Context, name string) (*entity.Role, error) {
	row, err := r.queries.GetRoleByName(ctx, name)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbRoleToEntity(row), nil
}

// GetUserRoles returns all roles held by a user.
func (r *RoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	rows, err := r.queries.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	roles := make([]*entity.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, dbRoleToEntity(row))
	}
	return roles, nil
}

// AssignRole grants a role to a user, recording who performed the assignment.
func (r *RoleRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error {
	var assignedByPg pgtype.UUID
	if assignedBy != nil {
		assignedByPg = pgtype.UUID{Bytes: *assignedBy, Valid: true}
	}
	err := r.queries.AssignRoleToUser(ctx, db.AssignRoleToUserParams{
		UserID:     userID,
		RoleID:     roleID,
		AssignedBy: assignedByPg,
	})
	return database.MapDBError(err)
}

// RemoveRole revokes a role from a user.
func (r *RoleRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	err := r.queries.RemoveRoleFromUser(ctx, db.RemoveRoleFromUserParams{
		UserID: userID,
		RoleID: roleID,
	})
	return database.MapDBError(err)
}

// UserHasRole checks whether a user holds a specific role (by role ID).
func (r *RoleRepository) UserHasRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error) {
	ok, err := r.queries.CheckUserHasRole(ctx, db.CheckUserHasRoleParams{
		UserID: userID,
		RoleID: roleID,
	})
	if err != nil {
		return false, database.MapDBError(err)
	}
	return ok, nil
}

// UserHasAnyRole checks whether a user holds at least one of the named roles.
func (r *RoleRepository) UserHasAnyRole(ctx context.Context, userID uuid.UUID, roleNames []string) (bool, error) {
	for _, name := range roleNames {
		role, err := r.queries.GetRoleByName(ctx, name)
		if err != nil {
			// Role doesn't exist — skip it rather than erroring.
			continue
		}
		ok, err := r.queries.CheckUserHasRole(ctx, db.CheckUserHasRoleParams{
			UserID: userID,
			RoleID: role.ID,
		})
		if err != nil {
			return false, database.MapDBError(err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func dbRoleToEntity(r db.UsrRole) *entity.Role {
	role := &entity.Role{
		ID:          r.ID,
		Name:        r.Name,
		Description: "",
		CreatedAt:   r.CreatedAt.Time,
	}
	if r.Description != nil {
		role.Description = *r.Description
	}
	if r.UpdatedAt.Valid {
		t := r.UpdatedAt.Time
		role.UpdatedAt = &t
	}
	return role
}
