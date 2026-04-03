package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/internal/validation"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

const bcryptCost = 12

// UserService handles all user business logic: account management and social profile.
type UserService struct {
	userRepo userRepo
	storage  storage.Storage
}

func NewUserService(userRepo *repository.UserRepository, storage storage.Storage) *UserService {
	return &UserService{userRepo: userRepo, storage: storage}
}

// --- Account management ---

func (s *UserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (uuid.UUID, error) {
	logger.Info(ctx, "creating user", "username", req.Username, "email", req.Email)

	if err := validateCreateRequest(req); err != nil {
		logger.Warn(ctx, "validation failed", "error", err)
		return uuid.Nil, err
	}

	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	displayName := strings.TrimSpace(req.DisplayName)

	usernameExists, err := s.userRepo.ExistsUsername(ctx, username)
	if err != nil {
		logger.LogError(ctx, err, "failed to check username existence")
		return uuid.Nil, errors.NewInternalError(err)
	}
	if usernameExists {
		return uuid.Nil, errors.NewConflictError("username already exists")
	}

	emailExists, err := s.userRepo.ExistsEmail(ctx, email)
	if err != nil {
		logger.LogError(ctx, err, "failed to check email existence")
		return uuid.Nil, errors.NewInternalError(err)
	}
	if emailExists {
		return uuid.Nil, errors.NewConflictError("email already exists")
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		logger.LogError(ctx, err, "failed to hash password")
		return uuid.Nil, errors.NewInternalError(err)
	}

	created, err := s.userRepo.CreateUser(ctx, &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		IsActive:     true,
		DisplayName:  displayName,
	})
	if err != nil {
		logger.LogError(ctx, err, "failed to create user")
		return uuid.Nil, err
	}

	logger.Info(ctx, "user created successfully", "user_id", created.ID)
	return created.ID, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	logger.Debug(ctx, "getting user by id", "user_id", id)

	u, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user", "user_id", id)
		return nil, err
	}

	if !u.IsActive {
		return nil, user.ErrAccountDisabled
	}

	return u, nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	if err := validation.ValidateRequired("username", username); err != nil {
		return nil, err
	}

	u, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user", "username", username)
		return nil, err
	}

	if !u.IsActive {
		return nil, user.ErrAccountDisabled
	}

	return u, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	if err := validation.ValidateRequired("email", email); err != nil {
		return nil, err
	}

	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user", "email", email)
		return nil, err
	}

	if !u.IsActive {
		return nil, user.ErrAccountDisabled
	}

	return u, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req *dto.UpdateUserRequest, updatedBy *uuid.UUID) (*entity.User, error) {
	logger.Info(ctx, "updating user", "user_id", id)

	if err := validateUpdateRequest(req); err != nil {
		return nil, err
	}

	existing, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user", "user_id", id)
		return nil, err
	}

	if !existing.IsActive {
		return nil, user.ErrAccountDisabled
	}

	var email *string
	if req.Email != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.Email))
		email = &normalized

		if normalized != existing.Email {
			emailExists, emailErr := s.userRepo.ExistsEmailExcludingUser(ctx, normalized, id)
			if emailErr != nil {
				return nil, errors.NewInternalError(emailErr)
			}
			if emailExists {
				return nil, errors.NewConflictError("email already exists")
			}
		}
	}

	updated, err := s.userRepo.UpdateUser(ctx, id, email, updatedBy)
	if err != nil {
		logger.LogError(ctx, err, "failed to update user", "user_id", id)
		return nil, err
	}

	logger.Info(ctx, "user updated successfully", "user_id", id)
	return updated, nil
}

func (s *UserService) DeactivateUser(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error {
	logger.Info(ctx, "deactivating user", "user_id", id)

	u, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user", "user_id", id)
		return err
	}

	if !u.IsActive {
		return user.ErrAccountDisabled
	}

	if err := s.userRepo.DeactivateUser(ctx, id, updatedBy); err != nil {
		logger.LogError(ctx, err, "failed to deactivate user", "user_id", id)
		return err
	}

	logger.Info(ctx, "user deactivated successfully", "user_id", id)
	return nil
}

func (s *UserService) BootstrapRootUser(ctx context.Context, email, password, username, displayName string) (bool, error) {
	existsRootUser, err := s.userRepo.ExistsUsername(ctx, username)
	if err != nil {
		return false, errors.NewInternalError(err)
	}
	if existsRootUser {
		logger.Info(ctx, "bootstrap skipped: users already exist", "username", username)
		return false, nil
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return false, errors.NewInternalError(err)
	}

	created, err := s.userRepo.CreateUser(ctx, &entity.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		IsActive:     true,
		DisplayName:  displayName,
	})
	if err != nil {
		logger.LogError(ctx, err, "bootstrap: failed to create root user")
		return false, errors.NewInternalError(err)
	}

	logger.Info(ctx, "bootstrap: root user created", "user_id", created.ID, "username", created.Username)
	return true, nil
}

// --- Social profile management ---

func (s *UserService) GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	u, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get profile", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}
	return u, nil
}

func (s *UserService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	u, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get profile", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}
	return u, nil
}

func (s *UserService) UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*entity.User, error) {
	updated, err := s.userRepo.UpdateUserProfile(ctx, userID, db.UpdateUserProfileParams{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		Website:     req.Website,
		Location:    req.Location,
	})
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to update profile", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	logger.Info(ctx, "profile updated", "user_id", userID)
	return updated, nil
}

func (s *UserService) UploadAvatar(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error) {
	existing, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user for avatar upload", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	newKey := fmt.Sprintf("avatars/%s%s", uuid.New().String(), ext)
	if err = s.storage.Put(ctx, newKey, r, size, contentType); err != nil {
		logger.LogError(ctx, err, "failed to upload avatar", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	oldKey := existing.AvatarKey
	updated, err := s.userRepo.UpdateUserProfile(ctx, userID, db.UpdateUserProfileParams{AvatarKey: &newKey})
	if err != nil {
		_ = s.storage.Delete(ctx, newKey)
		logger.LogError(ctx, err, "failed to update avatar key", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	if oldKey != nil {
		go func() { //nolint:gosec // fire-and-forget cleanup must outlive request context
			if delErr := s.storage.Delete(context.Background(), *oldKey); delErr != nil {
				logger.LogError(context.Background(), delErr, "failed to delete old avatar", "key", *oldKey)
			}
		}()
	}

	logger.Info(ctx, "avatar uploaded", "user_id", userID, "key", newKey)
	return updated, nil
}

func (s *UserService) UploadCover(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error) {
	existing, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, user.ErrUserNotFound
		}
		logger.LogError(ctx, err, "failed to get user for cover upload", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	newKey := fmt.Sprintf("covers/%s%s", uuid.New().String(), ext)
	if err = s.storage.Put(ctx, newKey, r, size, contentType); err != nil {
		logger.LogError(ctx, err, "failed to upload cover", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	oldKey := existing.CoverKey
	updated, err := s.userRepo.UpdateUserProfile(ctx, userID, db.UpdateUserProfileParams{CoverKey: &newKey})
	if err != nil {
		_ = s.storage.Delete(ctx, newKey)
		logger.LogError(ctx, err, "failed to update cover key", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	if oldKey != nil {
		go func() { //nolint:gosec // fire-and-forget cleanup must outlive request context
			if delErr := s.storage.Delete(context.Background(), *oldKey); delErr != nil {
				logger.LogError(context.Background(), delErr, "failed to delete old cover", "key", *oldKey)
			}
		}()
	}

	logger.Info(ctx, "cover uploaded", "user_id", userID, "key", newKey)
	return updated, nil
}
