package entity

import (
	"time"

	"github.com/google/uuid"
)

// Follow represents an asymmetric follow relationship (follower → followee).
type Follow struct {
	FollowerID uuid.UUID
	FolloweeID uuid.UUID
	CreatedAt  time.Time
}
