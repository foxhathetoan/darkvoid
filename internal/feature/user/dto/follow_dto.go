package dto

import (
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/pagination"
)

// FollowResponse is a single follow edge returned in list responses.
type FollowResponse struct {
	UserID    string `json:"user_id"`
	CreatedAt string `json:"followed_at"`
}

// FollowListResponse is the paginated list of followers or following.
type FollowListResponse struct {
	Data       []FollowResponse              `json:"data"`
	Pagination pagination.PaginationResponse `json:"pagination"`
}

// ToFollowerResponse converts a follow entity to a FollowResponse from the follower's perspective.
func ToFollowerResponse(f *entity.Follow) FollowResponse {
	return FollowResponse{
		UserID:    f.FollowerID.String(),
		CreatedAt: f.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// ToFollowingResponse converts a follow entity to a FollowResponse from the following's perspective.
func ToFollowingResponse(f *entity.Follow) FollowResponse {
	return FollowResponse{
		UserID:    f.FolloweeID.String(),
		CreatedAt: f.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
