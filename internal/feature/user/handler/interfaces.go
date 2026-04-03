package handler

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/pagination"
)

// authService defines the methods used by AuthHandler.
type authService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshAccessToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)
	Logout(ctx context.Context, req *dto.LogoutRequest) error
	LogoutAllSessions(ctx context.Context, userID uuid.UUID) error
	GetMe(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

// userService defines the methods used by UserHandler.
type userService interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *dto.UpdateUserRequest, updatedBy *uuid.UUID) (*entity.User, error)
	DeactivateUser(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error
}

// profileService defines the methods used by ProfileHandler.
type profileService interface {
	GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*entity.User, error)
	UploadAvatar(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error)
	UploadCover(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error)
}

// userResolver resolves a path parameter to a user UUID.
// When ?by=username, the param is treated as a username and looked up.
type userResolver interface {
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
}

// followService defines the methods used by FollowHandler.
type followService interface {
	Follow(ctx context.Context, followerID, followeeID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error
	GetFollowers(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error)
	GetFollowing(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error)
}

// followChecker defines the IsFollowing check used by ProfileHandler.
type followChecker interface {
	IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error)
}
