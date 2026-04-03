package entity

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user with both auth and social profile information.
type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash string
	IsActive     bool
	DisplayName  string
	Bio          *string
	AvatarKey    *string
	CoverKey     *string
	Website      *string
	Location     *string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
	CreatedBy    *uuid.UUID
	UpdatedBy    *uuid.UUID

	// Denormalized counters (maintained by DB triggers)
	FollowerCount  int64
	FollowingCount int64

	// Enriched fields (not stored, populated per-request)
	IsFollowing *bool // nil when viewer is unauthenticated or viewing own profile
}
