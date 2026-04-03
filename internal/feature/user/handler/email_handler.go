package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// emailService defines the methods used by EmailHandler.
type emailService interface {
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, email string) error
	SendPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

// EmailHandler handles HTTP requests for email-related operations.
type EmailHandler struct {
	emailService emailService
}

// NewEmailHandler creates a new email handler.
func NewEmailHandler(emailService emailService) *EmailHandler {
	return &EmailHandler{emailService: emailService}
}

// VerifyEmail godoc
//
//	@Summary		Verify email address
//	@Description	Verify a user's email address using the token sent via email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.VerifyEmailRequest		true	"Verification token"
//	@Success		200		{object}	httputil.MessageResponse	"Email verified successfully"
//	@Failure		400		{object}	errors.ErrorResponse		"Invalid or expired token"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				verifyEmail
//	@Router			/auth/verify-email [post]
func (h *EmailHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid request body"))
		return
	}

	if err := h.emailService.VerifyEmail(ctx, req.Token); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("Email verified successfully"))
}

// ResendVerification godoc
//
//	@Summary		Resend verification email
//	@Description	Resend the email verification link to the specified email address
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ResendVerificationRequest	true	"Email address"
//	@Success		200		{object}	httputil.MessageResponse		"Verification email sent if account exists"
//	@Failure		400		{object}	errors.ErrorResponse			"Invalid request"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				resendVerification
//	@Router			/auth/resend-verification [post]
func (h *EmailHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid request body"))
		return
	}

	if err := h.emailService.ResendVerification(ctx, req.Email); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	// Always return success to avoid leaking whether email exists
	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("If the email is registered, a verification link has been sent"))
}

// ForgotPassword godoc
//
//	@Summary		Request password reset
//	@Description	Send a password reset link to the specified email address
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ForgotPasswordRequest	true	"Email address"
//	@Success		200		{object}	httputil.MessageResponse	"Reset link sent if account exists"
//	@Failure		400		{object}	errors.ErrorResponse		"Invalid request"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				forgotPassword
//	@Router			/auth/forgot-password [post]
func (h *EmailHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid request body"))
		return
	}

	if err := h.emailService.SendPasswordReset(ctx, req.Email); err != nil {
		logger.LogError(ctx, err, "failed to send password reset email")
		errors.WriteJSON(w, err)
		return
	}

	// Always return success to avoid leaking whether email exists
	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("If the email is registered, a password reset link has been sent"))
}

// ResetPassword godoc
//
//	@Summary		Reset password
//	@Description	Reset the user's password using the token received via email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ResetPasswordRequest	true	"Reset token and new password"
//	@Success		200		{object}	httputil.MessageResponse	"Password reset successfully"
//	@Failure		400		{object}	errors.ErrorResponse		"Invalid or expired token"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				resetPassword
//	@Router			/auth/reset-password [post]
func (h *EmailHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid request body"))
		return
	}

	if err := h.emailService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("Password reset successfully"))
}
