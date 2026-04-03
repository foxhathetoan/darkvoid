package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
)

// userRepo defines the repository operations needed by UserService and AuthService.
type userRepo interface {
	ExistsUsername(ctx context.Context, username string) (bool, error)
	ExistsEmail(ctx context.Context, email string) (bool, error)
	ExistsEmailExcludingUser(ctx context.Context, email string, userID uuid.UUID) (bool, error)
	CreateUser(ctx context.Context, user *entity.User) (*entity.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, email *string, updatedBy *uuid.UUID) (*entity.User, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, params db.UpdateUserProfileParams) (*entity.User, error)
	UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string, updatedBy *uuid.UUID) error
	DeactivateUser(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error
}

// refreshTokenRepo defines the repository operations needed by RefreshTokenService.
type refreshTokenRepo interface {
	Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) (*entity.RefreshToken, error)
	GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error)
	Revoke(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
