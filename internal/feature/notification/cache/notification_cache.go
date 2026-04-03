package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const UnreadCountTTL = 10 * time.Minute

// NotificationCache abstracts caching for the notification feature.
// Currently caches unread count per user: notif:unread:{userID}  TTL 10m.
type NotificationCache interface {
	// GetUnreadCount returns the cached unread count.
	// Returns (-1, nil) on cache miss.
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)

	// SetUnreadCount stores the unread count for a user.
	SetUnreadCount(ctx context.Context, userID uuid.UUID, count int64) error

	// IncrementUnreadCount atomically increments the cached unread count.
	// No-op on cache miss (the next read will repopulate from DB).
	IncrementUnreadCount(ctx context.Context, userID uuid.UUID) error

	// InvalidateUnreadCount removes the cached unread count for a user.
	InvalidateUnreadCount(ctx context.Context, userID uuid.UUID) error
}
