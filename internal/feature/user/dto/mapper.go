package dto

import (
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// ToUserResponse converts entity.User to UserResponse DTO.
// s is used to build full public URLs from storage keys at response time.
func ToUserResponse(user *entity.User, s storage.Storage) UserResponse {
	resp := UserResponse{
		ID:             user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		IsActive:       user.IsActive,
		DisplayName:    user.DisplayName,
		Bio:            user.Bio,
		Website:        user.Website,
		Location:       user.Location,
		CreatedAt:      user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		FollowerCount:  user.FollowerCount,
		FollowingCount: user.FollowingCount,
		IsFollowing:    user.IsFollowing,
	}

	if user.AvatarKey != nil {
		u := s.URL(*user.AvatarKey)
		resp.AvatarURL = &u
	}
	if user.CoverKey != nil {
		u := s.URL(*user.CoverKey)
		resp.CoverURL = &u
	}
	if user.UpdatedAt != nil {
		t := user.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
		resp.UpdatedAt = &t
	}

	return resp
}

// ToProfileResponse converts entity.User to the public-facing ProfileResponse.
// Omits sensitive fields (email, is_active, updated_at).
func ToProfileResponse(user *entity.User, s storage.Storage) ProfileResponse {
	resp := ProfileResponse{
		ID:             user.ID.String(),
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		Bio:            user.Bio,
		Website:        user.Website,
		Location:       user.Location,
		JoinedAt:       user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		FollowerCount:  user.FollowerCount,
		FollowingCount: user.FollowingCount,
		IsFollowing:    user.IsFollowing,
	}
	if user.AvatarKey != nil {
		u := s.URL(*user.AvatarKey)
		resp.AvatarURL = &u
	}
	if user.CoverKey != nil {
		u := s.URL(*user.CoverKey)
		resp.CoverURL = &u
	}
	return resp
}
