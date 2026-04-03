package feed

import (
	"context"
	"time"

	"github.com/google/uuid"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

// PostReader defines the post data access methods needed by the feed context.
// Implemented at the app layer to avoid cross-context imports.
type PostReader interface {
	// GetFollowingPostsWithCursor fetches posts from followed authors using DB cursor pagination.
	// If cursor is nil, returns from the latest post with no time restriction.
	GetFollowingPostsWithCursor(ctx context.Context, authorIDs []uuid.UUID, cursor *FollowingCursor, limit int32) ([]*feedentity.Post, error)
	// GetTrendingPosts fetches top-liked public posts from the last 24 hours.
	GetTrendingPosts(ctx context.Context, limit int32) ([]*feedentity.Post, error)
	// GetDiscoverWithCursor fetches public posts for the discovery feed using cursor pagination.
	// If cursor is nil, returns from the latest post.
	GetDiscoverWithCursor(ctx context.Context, cursor *DiscoverCursor, limit int32, viewerID *uuid.UUID) ([]*feedentity.Post, error)
}

// MaxDiscoverTime is a sentinel used when no cursor is present — fetches from the far future so all posts are included.
var MaxDiscoverTime = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
