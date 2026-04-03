// Package cache provides caching abstractions for the post feature.
package cache

import (
	"context"
	"time"

	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
)

const (
	TrendingHashtagsTTL  = 15 * time.Minute
	HashtagPostsPage1TTL = 60 * time.Second
	HashtagSearchTTL     = 2 * time.Minute
)

// HashtagPostsPage holds the cached result of page 1 for a hashtag post listing.
// IsLiked is always false in this cache — viewer-specific enrichment is skipped.
type HashtagPostsPage struct {
	Posts      []*entity.Post `json:"posts"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// HashtagCache abstracts caching for hashtag data.
// Cache types:
//   - trending:hashtags           TTL 15m — top trending hashtag list, globally shared
//   - hashtag:posts:page1:{name}  TTL 60s — first page of posts for a hashtag
type HashtagCache interface {
	// GetTrendingHashtags returns the cached trending hashtag list.
	// Returns (nil, nil) on cache miss.
	GetTrendingHashtags(ctx context.Context) ([]*entity.TrendingHashtag, error)

	// SetTrendingHashtags stores the trending hashtag list.
	SetTrendingHashtags(ctx context.Context, tags []*entity.TrendingHashtag) error

	// InvalidateTrendingHashtags removes the trending hashtag cache.
	InvalidateTrendingHashtags(ctx context.Context) error

	// GetHashtagPostsPage1 returns the cached first page of posts for a hashtag.
	// Returns (nil, nil) on cache miss.
	GetHashtagPostsPage1(ctx context.Context, name string) (*HashtagPostsPage, error)

	// SetHashtagPostsPage1 caches the first page of posts for a hashtag.
	SetHashtagPostsPage1(ctx context.Context, name string, page *HashtagPostsPage) error

	// GetSearchResults returns cached prefix-search results for the given prefix.
	// Returns (nil, nil) on cache miss.
	GetSearchResults(ctx context.Context, prefix string) ([]string, error)

	// SetSearchResults caches prefix-search results.
	SetSearchResults(ctx context.Context, prefix string, names []string) error
}
