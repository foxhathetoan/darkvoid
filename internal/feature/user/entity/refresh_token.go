package entity

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token entity.
type RefreshToken struct {
	ID        uuid.UUID
	Token     string
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
	IsRevoked bool
}

// IsExpired checks if the refresh token has expired.
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the refresh token is valid (not expired and not revoked).
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsRevoked && !rt.IsExpired()
}
