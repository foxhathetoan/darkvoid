package cache

import (
	"context"

	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
)

// NopHashtagCache is a no-op implementation that always misses.
// Used when Redis is not configured.
type NopHashtagCache struct{}

// NewNopHashtagCache creates a new NopHashtagCache.
func NewNopHashtagCache() *NopHashtagCache { return &NopHashtagCache{} }

func (n *NopHashtagCache) GetTrendingHashtags(_ context.Context) ([]*entity.TrendingHashtag, error) {
	return nil, nil
}
func (n *NopHashtagCache) SetTrendingHashtags(_ context.Context, _ []*entity.TrendingHashtag) error {
	return nil
}
func (n *NopHashtagCache) InvalidateTrendingHashtags(_ context.Context) error { return nil }
func (n *NopHashtagCache) GetHashtagPostsPage1(_ context.Context, _ string) (*HashtagPostsPage, error) {
	return nil, nil //nolint:nilnil // cache miss by contract
}
func (n *NopHashtagCache) SetHashtagPostsPage1(_ context.Context, _ string, _ *HashtagPostsPage) error {
	return nil
}
func (n *NopHashtagCache) GetSearchResults(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (n *NopHashtagCache) SetSearchResults(_ context.Context, _ string, _ []string) error {
	return nil
}
