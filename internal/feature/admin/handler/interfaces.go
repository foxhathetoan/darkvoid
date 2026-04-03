package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/admin/dto"
	"github.com/jarviisha/darkvoid/internal/feature/admin/service"
)

// adminService is the narrow interface the handler depends on.
type adminService interface {
	// User management
	ListUsers(ctx context.Context, filter service.AdminListUsersFilter) (*dto.AdminListUsersResponse, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*dto.AdminUserResponse, error)
	SetUserActive(ctx context.Context, targetUserID uuid.UUID, isActive bool, adminID uuid.UUID) error

	// Role management
	ListRoles(ctx context.Context) (*dto.ListRolesResponse, error)
	CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleResponse, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) (*dto.UserRolesResponse, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, adminID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID, adminID uuid.UUID) error

	// Notifications
	SendNotificationToUser(ctx context.Context, adminID, targetUserID uuid.UUID, req *dto.AdminSendNotificationRequest) error
	BroadcastNotification(ctx context.Context, adminID uuid.UUID, req *dto.AdminSendNotificationRequest) (*dto.AdminBroadcastNotificationResponse, error)

	// Stats
	GetStats(ctx context.Context) (*dto.AdminStatsResponse, error)
}
