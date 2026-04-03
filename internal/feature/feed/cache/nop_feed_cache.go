package cache

import (
	"context"

	"github.com/google/uuid"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

// NopFeedCache is a no-op implementation that always misses.
// Used when Redis is not configured (falls back to pull on-the-fly).
type NopFeedCache struct{}

func NewNopFeedCache() *NopFeedCache { return &NopFeedCache{} }

func (n *NopFeedCache) GetFollowingIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (n *NopFeedCache) SetFollowingIDs(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}
func (n *NopFeedCache) InvalidateFollowingIDs(_ context.Context, _ uuid.UUID) error { return nil }
func (n *NopFeedCache) GetTrending(_ context.Context) ([]*feedentity.Post, error) {
	return nil, nil
}
func (n *NopFeedCache) SetTrending(_ context.Context, _ []*feedentity.Post) error { return nil }
func (n *NopFeedCache) InvalidateTrending(_ context.Context) error                { return nil }
