package feed

import (
	"context"

	"github.com/google/uuid"
)

// FollowReader resolves the list of user IDs that a given user follows.
// Implemented at the app layer to avoid cross-context imports.
type FollowReader interface {
	GetFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
