package feed

import (
	"context"

	"github.com/google/uuid"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

// UserReader resolves author information for a set of user IDs.
// Implemented at the app layer to avoid cross-context imports.
type UserReader interface {
	// GetAuthorsByIDs returns a map of userID → Author for all requested IDs.
	// Missing IDs (e.g. deleted users) are simply absent from the map.
	GetAuthorsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*feedentity.Author, error)
}
