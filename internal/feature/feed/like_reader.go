package feed

import (
	"context"

	"github.com/google/uuid"
)

// LikeReader provides like status lookups for feed enrichment.
type LikeReader interface {
	GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) ([]uuid.UUID, error)
}
