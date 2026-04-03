package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

const (
	RefreshTokenLength        = 32
	DefaultRefreshTokenExpiry = 7 * 24 * time.Hour
)

type RefreshTokenService struct {
	repo   refreshTokenRepo
	expiry time.Duration
}

func NewRefreshTokenService(repo *repository.RefreshTokenRepository) *RefreshTokenService {
	return &RefreshTokenService{repo: repo, expiry: DefaultRefreshTokenExpiry}
}

func NewRefreshTokenServiceWithExpiry(repo *repository.RefreshTokenRepository, expiry time.Duration) *RefreshTokenService {
	return &RefreshTokenService{repo: repo, expiry: expiry}
}

func (s *RefreshTokenService) GenerateToken(ctx context.Context, userID uuid.UUID) (*entity.RefreshToken, error) {
	tokenString, err := generateSecureToken()
	if err != nil {
		logger.LogError(ctx, err, "failed to generate secure token")
		return nil, errors.NewInternalError(err)
	}

	token, err := s.repo.Create(ctx, tokenString, userID, time.Now().Add(s.expiry))
	if err != nil {
		logger.LogError(ctx, err, "failed to create refresh token", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	logger.Info(ctx, "refresh token created", "user_id", userID, "token_id", token.ID)
	return token, nil
}

func (s *RefreshTokenService) ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error) {
	token, err := s.repo.GetByToken(ctx, tokenString)
	if err != nil {
		logger.Warn(ctx, "refresh token not found", "error", err)
		return uuid.Nil, errors.NewUnauthorizedError("invalid refresh token")
	}

	if !token.IsValid() {
		logger.Warn(ctx, "invalid refresh token", "token_id", token.ID)
		return uuid.Nil, errors.NewUnauthorizedError("refresh token is invalid or expired")
	}

	return token.UserID, nil
}

func (s *RefreshTokenService) RevokeToken(ctx context.Context, tokenString string) error {
	if err := s.repo.Revoke(ctx, tokenString); err != nil {
		logger.LogError(ctx, err, "failed to revoke refresh token")
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "refresh token revoked")
	return nil
}

func (s *RefreshTokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.RevokeAllUserTokens(ctx, userID); err != nil {
		logger.LogError(ctx, err, "failed to revoke all user tokens", "user_id", userID)
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "all user refresh tokens revoked", "user_id", userID)
	return nil
}

func (s *RefreshTokenService) CleanupExpiredTokens(ctx context.Context) error {
	if err := s.repo.DeleteExpired(ctx); err != nil {
		logger.LogError(ctx, err, "failed to cleanup expired tokens")
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "expired refresh tokens cleaned up")
	return nil
}

func (s *RefreshTokenService) GetExpiryDuration() time.Duration {
	return s.expiry
}

func generateSecureToken() (string, error) {
	b := make([]byte, RefreshTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
