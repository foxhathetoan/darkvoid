package dto

// CreateUserRequest represents user creation request.
type CreateUserRequest struct {
	Username    string `json:"username" example:"johndoe"`
	Email       string `json:"email" example:"john@example.com"`
	DisplayName string `json:"display_name" example:"John Doe"`
	Password    string `json:"password" example:"SecurePass123"`
}

// UpdateUserRequest represents user update request (auth fields only).
type UpdateUserRequest struct {
	Email *string `json:"email,omitempty" example:"john.updated@example.com"`
}

// UpdateProfileRequest represents fields the user can update on their profile.
// Avatar and cover images are updated via dedicated upload endpoints.
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name,omitempty" example:"Nguyen Van A"`
	Bio         *string `json:"bio,omitempty" example:"Software engineer & coffee lover"`
	Website     *string `json:"website,omitempty" example:"https://example.com"`
	Location    *string `json:"location,omitempty" example:"Ha Noi, Viet Nam"`
}

// UserResponse represents user data in API responses.
type UserResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username       string  `json:"username" example:"johndoe"`
	Email          string  `json:"email" example:"john@example.com"`
	IsActive       bool    `json:"is_active" example:"true"`
	DisplayName    string  `json:"display_name" example:"John Doe"`
	Bio            *string `json:"bio,omitempty"`
	AvatarURL      *string `json:"avatar_url,omitempty" example:"https://cdn.example.com/avatars/abc123.jpg"`
	CoverURL       *string `json:"cover_url,omitempty" example:"https://cdn.example.com/covers/def456.jpg"`
	Website        *string `json:"website,omitempty"`
	Location       *string `json:"location,omitempty"`
	CreatedAt      string  `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      *string `json:"updated_at,omitempty" example:"2024-01-02T00:00:00Z"`
	FollowerCount  int64   `json:"follower_count" example:"128"`
	FollowingCount int64   `json:"following_count" example:"64"`
	IsFollowing    *bool   `json:"is_following,omitempty"`
}

// ProfileResponse is the public-facing social profile representation.
// Unlike UserResponse, it omits sensitive fields (email, is_active).
type ProfileResponse struct {
	ID             string  `json:"id"`
	Username       string  `json:"username"`
	DisplayName    string  `json:"display_name"`
	Bio            *string `json:"bio,omitempty"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	CoverURL       *string `json:"cover_url,omitempty"`
	Website        *string `json:"website,omitempty"`
	Location       *string `json:"location,omitempty"`
	JoinedAt       string  `json:"joined_at"`
	FollowerCount  int64   `json:"follower_count"`
	FollowingCount int64   `json:"following_count"`
	IsFollowing    *bool   `json:"is_following,omitempty"`
}
