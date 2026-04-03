package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jarviisha/darkvoid/internal/feature/notification/entity"
)

// notifRepo defines the data access methods used by NotificationService.
type notifRepo interface {
	Create(ctx context.Context, recipientID, actorID uuid.UUID, notifType string, targetID, secondaryID *uuid.UUID, groupKey string) (*entity.Notification, error)
	CreateSystemNotification(ctx context.Context, recipientID, actorID uuid.UUID, message, groupKey string) (*entity.Notification, error)
	GetByRecipientWithCursor(ctx context.Context, recipientID uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorID uuid.UUID, limit int32) ([]*entity.Notification, error)
	MarkAsRead(ctx context.Context, notificationID, recipientID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) error
	CountUnread(ctx context.Context, recipientID uuid.UUID) (int64, error)
	GetGroupActors(ctx context.Context, recipientID uuid.UUID, groupKey string, limit int32) ([]uuid.UUID, int64, error)
	DeleteByActorAndGroupKey(ctx context.Context, actorID uuid.UUID, groupKey string) error
}

// userReader resolves user info for enriching notification actor data.
type userReader interface {
	GetAuthorsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.Actor, error)
}
