package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/pagination"
)

// userStore is the narrow interface the admin service needs from the user feature.
type userStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	AdminListUsers(ctx context.Context, filter AdminListUsersFilter) ([]*entity.User, error)
	AdminCountUsers(ctx context.Context, filter AdminListUsersFilter) (int64, error)
	AdminSetUserActive(ctx context.Context, id uuid.UUID, isActive bool, updatedBy uuid.UUID) error
	ListAllActiveUserIDs(ctx context.Context) ([]uuid.UUID, error)
}

// notifEmitter is the narrow interface the admin service needs to send notifications.
type notifEmitter interface {
	EmitSystemAnnouncement(ctx context.Context, adminID, recipientID uuid.UUID, message, groupKey string) error
}

// roleStore is the narrow interface the admin service needs for role management.
type roleStore interface {
	ListRoles(ctx context.Context) ([]*entity.Role, error)
	CreateRole(ctx context.Context, name string, description *string) (*entity.Role, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*entity.Role, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	UserHasAnyRole(ctx context.Context, userID uuid.UUID, roleNames []string) (bool, error)
}

// AdminListUsersFilter holds the filter/pagination options for listing users.
type AdminListUsersFilter struct {
	pagination.PaginationRequest
	Query    *string
	IsActive *bool
}
