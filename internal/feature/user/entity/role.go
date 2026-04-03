package entity

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a user role for authorization.
type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}
