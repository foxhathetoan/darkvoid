package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/admin/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// AdminService implements admin operations across users and roles.
type AdminService struct {
	userStore    userStore
	roleStore    roleStore
	storage      storage.Storage
	notifEmitter notifEmitter // optional, nil = notifications disabled
}

// NewAdminService creates an AdminService with the required dependencies.
func NewAdminService(userStore userStore, roleStore roleStore, store storage.Storage) *AdminService {
	return &AdminService{
		userStore: userStore,
		roleStore: roleStore,
		storage:   store,
	}
}

// WithNotificationEmitter attaches a notification emitter. Called at wire-up time.
func (s *AdminService) WithNotificationEmitter(e notifEmitter) {
	s.notifEmitter = e
}

// ─── User Management ─────────────────────────────────────────────────────────

// ListUsers returns a paginated list of all users matching the given filter.
func (s *AdminService) ListUsers(ctx context.Context, filter AdminListUsersFilter) (*dto.AdminListUsersResponse, error) {
	users, err := s.userStore.AdminListUsers(ctx, filter)
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to list users")
		return nil, errors.NewInternalError(err)
	}

	total, err := s.userStore.AdminCountUsers(ctx, filter)
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to count users")
		return nil, errors.NewInternalError(err)
	}

	data := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		data = append(data, toAdminUserResponse(u, s.storage))
	}

	return &dto.AdminListUsersResponse{
		Data:               data,
		PaginationResponse: pagination.NewPaginationResponse(total, filter.Limit, filter.Offset),
	}, nil
}

// GetUser returns the admin view of a single user.
func (s *AdminService) GetUser(ctx context.Context, userID uuid.UUID) (*dto.AdminUserResponse, error) {
	u, err := s.userStore.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := toAdminUserResponse(u, s.storage)
	return &resp, nil
}

// SetUserActive activates or deactivates a user account.
func (s *AdminService) SetUserActive(ctx context.Context, targetUserID uuid.UUID, isActive bool, adminID uuid.UUID) error {
	if err := s.userStore.AdminSetUserActive(ctx, targetUserID, isActive, adminID); err != nil {
		logger.LogError(ctx, err, "admin: failed to set user active status",
			"target_user_id", targetUserID,
			"is_active", isActive,
		)
		return fmt.Errorf("set user active %s: %w", targetUserID, err)
	}
	logger.Info(ctx, "admin: user status updated",
		"target_user_id", targetUserID,
		"is_active", isActive,
		"admin_id", adminID,
	)
	return nil
}

// ─── Role Management ─────────────────────────────────────────────────────────

// ListRoles returns all roles in the system.
func (s *AdminService) ListRoles(ctx context.Context) (*dto.ListRolesResponse, error) {
	roles, err := s.roleStore.ListRoles(ctx)
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to list roles")
		return nil, errors.NewInternalError(err)
	}
	data := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		data = append(data, toRoleResponse(r))
	}
	return &dto.ListRolesResponse{Data: data}, nil
}

// CreateRole creates a new role.
func (s *AdminService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	if req.Name == "" {
		return nil, errors.NewBadRequestError("role name is required")
	}

	role, err := s.roleStore.CreateRole(ctx, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, errors.ErrConflict) {
			return nil, errors.NewConflictError("role already exists")
		}
		logger.LogError(ctx, err, "admin: failed to create role", "name", req.Name)
		return nil, errors.NewInternalError(err)
	}

	logger.Info(ctx, "admin: role created", "role_id", role.ID, "name", role.Name)
	resp := toRoleResponse(role)
	return &resp, nil
}

// GetUserRoles returns the roles held by a user.
func (s *AdminService) GetUserRoles(ctx context.Context, userID uuid.UUID) (*dto.UserRolesResponse, error) {
	roles, err := s.roleStore.GetUserRoles(ctx, userID)
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to get user roles", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}
	data := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		data = append(data, toRoleResponse(r))
	}
	return &dto.UserRolesResponse{UserID: userID.String(), Roles: data}, nil
}

// AssignRole grants a role to a user.
func (s *AdminService) AssignRole(ctx context.Context, userID, roleID uuid.UUID, adminID uuid.UUID) error {
	if _, err := s.userStore.GetUserByID(ctx, userID); err != nil {
		return err
	}
	if _, err := s.roleStore.GetRoleByID(ctx, roleID); err != nil {
		return err
	}

	if err := s.roleStore.AssignRole(ctx, userID, roleID, &adminID); err != nil {
		if errors.Is(err, errors.ErrConflict) {
			return errors.NewConflictError("user already has this role")
		}
		logger.LogError(ctx, err, "admin: failed to assign role",
			"user_id", userID,
			"role_id", roleID,
		)
		return errors.NewInternalError(err)
	}

	logger.Info(ctx, "admin: role assigned",
		"user_id", userID,
		"role_id", roleID,
		"admin_id", adminID,
	)
	return nil
}

// RemoveRole revokes a role from a user.
func (s *AdminService) RemoveRole(ctx context.Context, userID, roleID uuid.UUID, adminID uuid.UUID) error {
	if err := s.roleStore.RemoveRole(ctx, userID, roleID); err != nil {
		logger.LogError(ctx, err, "admin: failed to remove role",
			"user_id", userID,
			"role_id", roleID,
		)
		return fmt.Errorf("remove role %s from user %s: %w", roleID, userID, err)
	}
	logger.Info(ctx, "admin: role removed",
		"user_id", userID,
		"role_id", roleID,
		"admin_id", adminID,
	)
	return nil
}

// ─── RBAC helper — implements middleware.RoleChecker ─────────────────────────

// UserHasAnyRole checks whether a user holds at least one of the named roles.
func (s *AdminService) UserHasAnyRole(ctx context.Context, userID uuid.UUID, roleNames []string) (bool, error) {
	return s.roleStore.UserHasAnyRole(ctx, userID, roleNames)
}

// ─── Stats ────────────────────────────────────────────────────────────────────

// GetStats returns basic platform statistics.
func (s *AdminService) GetStats(ctx context.Context) (*dto.AdminStatsResponse, error) {
	activeFlag := true
	inactiveFlag := false

	total, err := s.userStore.AdminCountUsers(ctx, AdminListUsersFilter{})
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to count total users")
		return nil, errors.NewInternalError(err)
	}

	active, err := s.userStore.AdminCountUsers(ctx, AdminListUsersFilter{IsActive: &activeFlag})
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to count active users")
		return nil, errors.NewInternalError(err)
	}

	inactive, err := s.userStore.AdminCountUsers(ctx, AdminListUsersFilter{IsActive: &inactiveFlag})
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to count inactive users")
		return nil, errors.NewInternalError(err)
	}

	roles, err := s.roleStore.ListRoles(ctx)
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to count roles")
		return nil, errors.NewInternalError(err)
	}

	return &dto.AdminStatsResponse{
		TotalUsers:    total,
		ActiveUsers:   active,
		InactiveUsers: inactive,
		TotalRoles:    int64(len(roles)),
	}, nil
}

// ─── Notification Management ──────────────────────────────────────────────────

// SendNotificationToUser sends a system announcement to a single user.
func (s *AdminService) SendNotificationToUser(ctx context.Context, adminID, targetUserID uuid.UUID, req *dto.AdminSendNotificationRequest) error {
	if s.notifEmitter == nil {
		return errors.New("NOTIFICATIONS_DISABLED", "notification service not configured", 503)
	}
	if req.Message == "" {
		return errors.NewBadRequestError("message is required")
	}
	if _, err := s.userStore.GetUserByID(ctx, targetUserID); err != nil {
		return err
	}
	groupKey := fmt.Sprintf("system:%s", uuid.New().String())
	if err := s.notifEmitter.EmitSystemAnnouncement(ctx, adminID, targetUserID, req.Message, groupKey); err != nil {
		logger.LogError(ctx, err, "admin: failed to send notification to user",
			"admin_id", adminID, "target_user_id", targetUserID)
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "admin: notification sent to user", "admin_id", adminID, "target_user_id", targetUserID)
	return nil
}

// BroadcastNotification sends a system announcement to all active users.
// Errors per-user are logged and skipped; the method returns the count of successful sends.
func (s *AdminService) BroadcastNotification(ctx context.Context, adminID uuid.UUID, req *dto.AdminSendNotificationRequest) (*dto.AdminBroadcastNotificationResponse, error) {
	if s.notifEmitter == nil {
		return nil, errors.New("NOTIFICATIONS_DISABLED", "notification service not configured", 503)
	}
	if req.Message == "" {
		return nil, errors.NewBadRequestError("message is required")
	}

	userIDs, err := s.userStore.ListAllActiveUserIDs(ctx)
	if err != nil {
		logger.LogError(ctx, err, "admin: failed to list active users for broadcast")
		return nil, errors.NewInternalError(err)
	}

	// One group_key shared across all recipients of this broadcast so it can be
	// identified as a single event; uniqueness per recipient comes from recipient_id.
	groupKey := fmt.Sprintf("system:%s", uuid.New().String())

	var sent int
	for _, userID := range userIDs {
		// Skip sending to the admin themselves.
		if userID == adminID {
			continue
		}
		if err := s.notifEmitter.EmitSystemAnnouncement(ctx, adminID, userID, req.Message, groupKey); err != nil {
			logger.LogError(ctx, err, "admin: broadcast notification failed for user", "user_id", userID)
			continue
		}
		sent++
	}

	logger.Info(ctx, "admin: broadcast notification sent", "admin_id", adminID, "sent", sent, "total", len(userIDs))
	return &dto.AdminBroadcastNotificationResponse{SentCount: sent}, nil
}

// ─── Private helpers ─────────────────────────────────────────────────────────

func toAdminUserResponse(u *entity.User, s storage.Storage) dto.AdminUserResponse {
	resp := dto.AdminUserResponse{
		ID:             u.ID.String(),
		Username:       u.Username,
		Email:          u.Email,
		DisplayName:    u.DisplayName,
		IsActive:       u.IsActive,
		FollowerCount:  u.FollowerCount,
		FollowingCount: u.FollowingCount,
		CreatedAt:      u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if u.AvatarKey != nil {
		url := s.URL(*u.AvatarKey)
		resp.AvatarURL = &url
	}
	if u.UpdatedAt != nil {
		t := u.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.UpdatedAt = &t
	}
	return resp
}

func toRoleResponse(r *entity.Role) dto.RoleResponse {
	resp := dto.RoleResponse{
		ID:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if r.UpdatedAt != nil {
		t := r.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.UpdatedAt = &t
	}
	return resp
}
