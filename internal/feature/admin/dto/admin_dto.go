// Package dto
package dto

import "github.com/jarviisha/darkvoid/internal/pagination"

// ─── User Management ─────────────────────────────────────────────────────────

// AdminUserResponse is the admin view of a user — includes sensitive fields
// not exposed in the public UserResponse.
type AdminUserResponse struct {
	ID             string  `json:"id"              example:"550e8400-e29b-41d4-a716-446655440000"`
	Username       string  `json:"username"        example:"johndoe"`
	Email          string  `json:"email"           example:"john@example.com"`
	DisplayName    string  `json:"display_name"    example:"John Doe"`
	IsActive       bool    `json:"is_active"       example:"true"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	FollowerCount  int64   `json:"follower_count"  example:"120"`
	FollowingCount int64   `json:"following_count" example:"80"`
	CreatedAt      string  `json:"created_at"      example:"2024-01-15T10:30:00Z"`
	UpdatedAt      *string `json:"updated_at,omitempty"`
}

// AdminListUsersResponse wraps a paginated list of admin user views.
type AdminListUsersResponse struct {
	Data []AdminUserResponse `json:"data"`
	pagination.PaginationResponse
}

// AdminSetUserStatusRequest activates or deactivates a user account.
type AdminSetUserStatusRequest struct {
	IsActive bool `json:"is_active" example:"false"`
}

// ─── Role Management ─────────────────────────────────────────────────────────

// RoleResponse is the public representation of a role.
type RoleResponse struct {
	ID          string  `json:"id"                   example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string  `json:"name"                 example:"moderator"`
	Description string  `json:"description"          example:"Can moderate content"`
	CreatedAt   string  `json:"created_at"           example:"2024-01-15T10:30:00Z"`
	UpdatedAt   *string `json:"updated_at,omitempty" example:"2024-02-01T08:00:00Z"`
}

// ListRolesResponse wraps a list of roles.
type ListRolesResponse struct {
	Data []RoleResponse `json:"data"`
}

// CreateRoleRequest defines the payload for creating a new role.
type CreateRoleRequest struct {
	Name        string  `json:"name"        example:"moderator"`
	Description *string `json:"description" example:"Can moderate content"`
}

// AssignRoleRequest defines the payload for assigning a role to a user.
type AssignRoleRequest struct {
	RoleID string `json:"role_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UserRolesResponse wraps the roles held by a specific user.
type UserRolesResponse struct {
	UserID string         `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Roles  []RoleResponse `json:"roles"`
}

// ─── Notifications ────────────────────────────────────────────────────────────

// AdminSendNotificationRequest is the payload for sending a system notification.
type AdminSendNotificationRequest struct {
	Message string `json:"message" example:"Scheduled maintenance on 2026-04-01 at 02:00 UTC"`
}

// AdminBroadcastNotificationResponse reports how many users received the broadcast.
type AdminBroadcastNotificationResponse struct {
	SentCount int `json:"sent_count" example:"980"`
}

// ─── Stats ────────────────────────────────────────────────────────────────────

// AdminStatsResponse provides basic platform statistics.
type AdminStatsResponse struct {
	TotalUsers    int64 `json:"total_users"         example:"1024"`
	ActiveUsers   int64 `json:"active_users"        example:"980"`
	InactiveUsers int64 `json:"inactive_users"      example:"44"`
	TotalPosts    int64 `json:"total_posts"         example:"5200"`
	TotalRoles    int64 `json:"total_roles"         example:"3"`
}
