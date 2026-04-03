package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/notification"
	"github.com/jarviisha/darkvoid/internal/feature/notification/broker"
	"github.com/jarviisha/darkvoid/internal/feature/notification/entity"
)

// notifService defines the methods used by NotificationHandler.
type notifService interface {
	GetNotifications(ctx context.Context, userID uuid.UUID, cursor *notification.NotificationCursor) ([]*entity.Notification, *notification.NotificationCursor, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}

// sseBroker defines the methods used by the SSE stream handler.
type sseBroker interface {
	Subscribe(ctx context.Context, userID uuid.UUID) (<-chan broker.Event, func())
}
