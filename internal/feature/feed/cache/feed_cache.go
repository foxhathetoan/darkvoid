package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

const (
	FollowingIDsTTL = 5 * time.Minute
	TrendingTTL     = 15 * time.Minute
)

// FeedCache abstracts caching for the feed feature.
// Two cache types:
//   - Per-user following IDs: following:ids:{userID}  TTL 5m  — []uuid.UUID of followed authors
//   - Global trending:        trending:posts           TTL 15m — raw []Post, shared across all users
type FeedCache interface {
	// GetFollowingIDs returns the cached list of author IDs a user follows.
	// Returns (nil, nil) on cache miss.
	GetFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)

	// SetFollowingIDs stores the list of followed author IDs for a user.
	SetFollowingIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error

	// InvalidateFollowingIDs removes the cached following IDs for a user.
	// Called when the user follows or unfollows someone.
	InvalidateFollowingIDs(ctx context.Context, userID uuid.UUID) error

	// GetTrending returns the globally cached trending posts.
	// Returns (nil, nil) on cache miss.
	GetTrending(ctx context.Context) ([]*feedentity.Post, error)

	// SetTrending stores the global trending post list.
	SetTrending(ctx context.Context, posts []*feedentity.Post) error
	// InvalidateTrending removes the trending post cache.
	// Called when a like/unlike changes post scores.
	InvalidateTrending(ctx context.Context) error
}
