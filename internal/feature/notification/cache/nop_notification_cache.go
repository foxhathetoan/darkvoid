package cache

import (
	"context"

	"github.com/google/uuid"
)

// NopNotificationCache is a no-op implementation that always misses.
// Used when Redis is not configured.
type NopNotificationCache struct{}

func NewNopNotificationCache() *NopNotificationCache { return &NopNotificationCache{} }

func (n *NopNotificationCache) GetUnreadCount(_ context.Context, _ uuid.UUID) (int64, error) {
	return -1, nil
}
func (n *NopNotificationCache) SetUnreadCount(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}
func (n *NopNotificationCache) IncrementUnreadCount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (n *NopNotificationCache) InvalidateUnreadCount(_ context.Context, _ uuid.UUID) error {
	return nil
}
