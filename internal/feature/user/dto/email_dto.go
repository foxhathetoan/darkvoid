package dto

// VerifyEmailRequest represents a request to verify an email address.
type VerifyEmailRequest struct {
	Token string `json:"token" example:"abc123def456"`
}

// ResendVerificationRequest represents a request to resend the verification email.
type ResendVerificationRequest struct {
	Email string `json:"email" example:"john@example.com"`
}

// ForgotPasswordRequest represents a request to initiate a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email" example:"john@example.com"`
}

// ResetPasswordRequest represents a request to reset the password using a token.
type ResetPasswordRequest struct {
	Token       string `json:"token" example:"abc123def456"`
	NewPassword string `json:"new_password" example:"NewSecurePass123"`
}
