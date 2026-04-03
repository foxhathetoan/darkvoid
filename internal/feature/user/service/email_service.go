package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/mailer"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

const (
	verifyTokenExpiry = 24 * time.Hour
	resetTokenExpiry  = 1 * time.Hour
)

// emailTokenRepo defines the repository operations needed by EmailService.
type emailTokenRepo interface {
	Create(ctx context.Context, userID uuid.UUID, token string, tokenType entity.EmailTokenType, expiresAt time.Time) (*entity.EmailToken, error)
	GetByToken(ctx context.Context, token string) (*entity.EmailToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType entity.EmailTokenType) error
}

// EmailService handles sending emails and managing email tokens.
type EmailService struct {
	mailer    mailer.Mailer
	templates *mailer.Templates
	tokenRepo emailTokenRepo
	userRepo  userRepo
	baseURL   string
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	m mailer.Mailer,
	templates *mailer.Templates,
	tokenRepo emailTokenRepo,
	userRepo userRepo,
	baseURL string,
) *EmailService {
	return &EmailService{
		mailer:    m,
		templates: templates,
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		baseURL:   baseURL,
	}
}

// SendWelcome sends a welcome email to the user. Errors are logged, not propagated.
func (s *EmailService) SendWelcome(ctx context.Context, email, username string) {
	html, err := s.templates.RenderWelcome(mailer.WelcomeData{
		Username: username,
	})
	if err != nil {
		logger.LogError(ctx, err, "failed to render welcome email template")
		return
	}

	if err := s.mailer.Send(ctx, &mailer.Message{
		To:      []string{email},
		Subject: "Welcome to DarkVoid",
		HTML:    html,
		Text:    fmt.Sprintf("Hi %s, welcome to DarkVoid! Your account has been created successfully.", username),
	}); err != nil {
		logger.LogError(ctx, err, "failed to send welcome email", "email", email)
	}
}

// SendVerification creates a verification token and sends a verification email.
// Errors are logged, not propagated (fire-and-forget side effect).
func (s *EmailService) SendVerification(ctx context.Context, userID uuid.UUID, email, username string) {
	// Clean up any existing verification tokens for this user
	if err := s.tokenRepo.DeleteByUserAndType(ctx, userID, entity.EmailTokenVerify); err != nil {
		logger.LogError(ctx, err, "failed to delete old verification tokens", "user_id", userID)
	}

	token, err := generateSecureToken()
	if err != nil {
		logger.LogError(ctx, err, "failed to generate verification token")
		return
	}

	_, err = s.tokenRepo.Create(ctx, userID, token, entity.EmailTokenVerify, time.Now().Add(verifyTokenExpiry))
	if err != nil {
		logger.LogError(ctx, err, "failed to create verification token", "user_id", userID)
		return
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)
	html, err := s.templates.RenderVerifyEmail(mailer.VerifyEmailData{
		Username:  username,
		VerifyURL: verifyURL,
		ExpiresIn: "24 hours",
	})
	if err != nil {
		logger.LogError(ctx, err, "failed to render verification email template")
		return
	}

	if err := s.mailer.Send(ctx, &mailer.Message{
		To:      []string{email},
		Subject: "Verify your email - DarkVoid",
		HTML:    html,
		Text:    fmt.Sprintf("Hi %s, please verify your email by visiting: %s", username, verifyURL),
	}); err != nil {
		logger.LogError(ctx, err, "failed to send verification email", "email", email)
	}
}

// VerifyEmail validates a verification token and marks the associated user's email as verified.
func (s *EmailService) VerifyEmail(ctx context.Context, tokenStr string) error {
	if tokenStr == "" {
		return errors.NewBadRequestError("token is required")
	}

	token, err := s.tokenRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		return errors.NewBadRequestError("invalid or expired token")
	}

	if token.Type != entity.EmailTokenVerify {
		return errors.NewBadRequestError("invalid token type")
	}

	if token.IsUsed() {
		return errors.NewBadRequestError("token has already been used")
	}

	if token.IsExpired() {
		return errors.NewBadRequestError("token has expired")
	}

	if err := s.tokenRepo.MarkUsed(ctx, token.ID); err != nil {
		logger.LogError(ctx, err, "failed to mark verification token as used")
		return errors.NewInternalError(err)
	}

	logger.Info(ctx, "email verified successfully", "user_id", token.UserID)
	return nil
}

// ResendVerification re-sends a verification email for the given email address.
func (s *EmailService) ResendVerification(ctx context.Context, email string) error {
	if email == "" {
		return errors.NewBadRequestError("email is required")
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Do not leak whether the email exists
		logger.Warn(ctx, "resend verification for unknown email", "email", email)
		return nil
	}

	s.SendVerification(ctx, user.ID, user.Email, user.Username)
	return nil
}

// SendPasswordReset creates a reset token and sends a password reset email.
func (s *EmailService) SendPasswordReset(ctx context.Context, email string) error {
	if email == "" {
		return errors.NewBadRequestError("email is required")
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Do not leak whether the email exists
		logger.Warn(ctx, "password reset for unknown email", "email", email)
		return nil
	}

	// Clean up any existing reset tokens for this user
	if err = s.tokenRepo.DeleteByUserAndType(ctx, user.ID, entity.EmailTokenResetPassword); err != nil {
		logger.LogError(ctx, err, "failed to delete old reset tokens", "user_id", user.ID)
	}

	token, err := generateSecureToken()
	if err != nil {
		logger.LogError(ctx, err, "failed to generate reset token")
		return errors.NewInternalError(err)
	}

	_, err = s.tokenRepo.Create(ctx, user.ID, token, entity.EmailTokenResetPassword, time.Now().Add(resetTokenExpiry))
	if err != nil {
		logger.LogError(ctx, err, "failed to create reset token", "user_id", user.ID)
		return errors.NewInternalError(err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	html, err := s.templates.RenderResetPassword(mailer.ResetPasswordData{
		Username:  user.Username,
		ResetURL:  resetURL,
		ExpiresIn: "1 hour",
	})
	if err != nil {
		logger.LogError(ctx, err, "failed to render reset password email template")
		return errors.NewInternalError(err)
	}

	if err := s.mailer.Send(ctx, &mailer.Message{
		To:      []string{user.Email},
		Subject: "Reset your password - DarkVoid",
		HTML:    html,
		Text:    fmt.Sprintf("Hi %s, reset your password by visiting: %s", user.Username, resetURL),
	}); err != nil {
		logger.LogError(ctx, err, "failed to send password reset email", "email", user.Email)
		return errors.NewInternalError(err)
	}

	logger.Info(ctx, "password reset email sent", "email", user.Email)
	return nil
}

// ResetPassword validates a reset token and sets the new password.
func (s *EmailService) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	if tokenStr == "" {
		return errors.NewBadRequestError("token is required")
	}
	if newPassword == "" {
		return errors.NewBadRequestError("new password is required")
	}

	token, err := s.tokenRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		return errors.NewBadRequestError("invalid or expired token")
	}

	if token.Type != entity.EmailTokenResetPassword {
		return errors.NewBadRequestError("invalid token type")
	}

	if token.IsUsed() {
		return errors.NewBadRequestError("token has already been used")
	}

	if token.IsExpired() {
		return errors.NewBadRequestError("token has expired")
	}

	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return errors.NewInternalError(err)
	}

	if err := s.userRepo.UpdateUserPassword(ctx, token.UserID, hashedPassword, nil); err != nil {
		return errors.NewInternalError(err)
	}

	if err := s.tokenRepo.MarkUsed(ctx, token.ID); err != nil {
		logger.LogError(ctx, err, "failed to mark reset token as used")
	}

	logger.Info(ctx, "password reset successfully", "user_id", token.UserID)
	return nil
}
