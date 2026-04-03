package entity

import (
	"time"

	"github.com/google/uuid"
)

// EmailTokenType represents the purpose of an email token.
type EmailTokenType string

const (
	EmailTokenVerify        EmailTokenType = "verify_email"
	EmailTokenResetPassword EmailTokenType = "reset_password"
)

// EmailToken represents a one-time token sent via email for verification or password reset.
type EmailToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	Type      EmailTokenType
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsExpired reports whether the token has passed its expiration time.
func (t *EmailToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed reports whether the token has already been redeemed.
func (t *EmailToken) IsUsed() bool {
	return t.UsedAt != nil
}
