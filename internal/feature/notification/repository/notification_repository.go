package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/notification/db"
	"github.com/jarviisha/darkvoid/internal/feature/notification/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

type NotificationRepository struct {
	queries *db.Queries
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{queries: db.New(pool)}
}

func (r *NotificationRepository) Create(ctx context.Context, recipientID, actorID uuid.UUID, notifType string, targetID, secondaryID *uuid.UUID, groupKey string) (*entity.Notification, error) {
	row, err := r.queries.CreateNotification(ctx, db.CreateNotificationParams{
		RecipientID: recipientID,
		ActorID:     actorID,
		Type:        notifType,
		TargetID:    uuidToNullable(targetID),
		SecondaryID: uuidToNullable(secondaryID),
		GroupKey:    groupKey,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return createRowToNotification(row), nil
}

func (r *NotificationRepository) GetByRecipientWithCursor(ctx context.Context, recipientID uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorID uuid.UUID, limit int32) ([]*entity.Notification, error) {
	rows, err := r.queries.GetNotificationsWithCursor(ctx, db.GetNotificationsWithCursorParams{
		RecipientID: recipientID,
		Column2:     cursorCreatedAt,
		Column3:     cursorID,
		Limit:       limit,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return cursorRowsToNotifications(rows), nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, notificationID, recipientID uuid.UUID) error {
	return database.MapDBError(r.queries.MarkAsRead(ctx, db.MarkAsReadParams{
		ID:          notificationID,
		RecipientID: recipientID,
	}))
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) error {
	return database.MapDBError(r.queries.MarkAllAsRead(ctx, recipientID))
}

func (r *NotificationRepository) CountUnread(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	count, err := r.queries.CountUnread(ctx, recipientID)
	if err != nil {
		return 0, database.MapDBError(err)
	}
	return count, nil
}

func (r *NotificationRepository) GetGroupActors(ctx context.Context, recipientID uuid.UUID, groupKey string, limit int32) ([]uuid.UUID, int64, error) {
	rows, err := r.queries.GetGroupActors(ctx, db.GetGroupActorsParams{
		RecipientID: recipientID,
		GroupKey:    groupKey,
		Limit:       limit,
	})
	if err != nil {
		return nil, 0, database.MapDBError(err)
	}
	if len(rows) == 0 {
		return nil, 0, nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ActorID
	}
	return ids, rows[0].TotalCount, nil
}

func (r *NotificationRepository) DeleteByActorAndGroupKey(ctx context.Context, actorID uuid.UUID, groupKey string) error {
	return database.MapDBError(r.queries.DeleteByActorAndGroupKey(ctx, db.DeleteByActorAndGroupKeyParams{
		ActorID:  actorID,
		GroupKey: groupKey,
	}))
}

func (r *NotificationRepository) CreateSystemNotification(ctx context.Context, recipientID, actorID uuid.UUID, message, groupKey string) (*entity.Notification, error) {
	row, err := r.queries.CreateSystemNotification(ctx, db.CreateSystemNotificationParams{
		RecipientID: recipientID,
		ActorID:     actorID,
		Type:        string(entity.TypeSystemAnnouncement),
		GroupKey:    groupKey,
		Message:     &message,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return fullRowToNotification(row), nil
}

// --- helpers ---

func fullRowToNotification(row db.NotificationNotification) *entity.Notification {
	return &entity.Notification{
		ID:          row.ID,
		RecipientID: row.RecipientID,
		ActorID:     row.ActorID,
		Type:        entity.NotificationType(row.Type),
		Payload:     buildPayload(row.Type, row.TargetID, row.SecondaryID, row.Message),
		GroupKey:    row.GroupKey,
		IsRead:      row.IsRead,
		CreatedAt:   row.CreatedAt.Time,
	}
}

func createRowToNotification(row db.CreateNotificationRow) *entity.Notification {
	return &entity.Notification{
		ID:          row.ID,
		RecipientID: row.RecipientID,
		ActorID:     row.ActorID,
		Type:        entity.NotificationType(row.Type),
		Payload:     buildPayload(row.Type, row.TargetID, row.SecondaryID, nil),
		GroupKey:    row.GroupKey,
		IsRead:      row.IsRead,
		CreatedAt:   row.CreatedAt.Time,
	}
}

func cursorRowsToNotifications(rows []db.GetNotificationsWithCursorRow) []*entity.Notification {
	result := make([]*entity.Notification, len(rows))
	for i, row := range rows {
		result[i] = &entity.Notification{
			ID:          row.ID,
			RecipientID: row.RecipientID,
			ActorID:     row.ActorID,
			Type:        entity.NotificationType(row.Type),
			Payload:     buildPayload(row.Type, row.TargetID, row.SecondaryID, nil),
			GroupKey:    row.GroupKey,
			IsRead:      row.IsRead,
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return result
}

// buildPayload maps raw DB columns to a typed Payload based on notification type.
func buildPayload(notifType string, targetID, secondaryID pgtype.UUID, message *string) entity.Payload {
	getTargetID := func() uuid.UUID {
		if targetID.Valid {
			return uuid.UUID(targetID.Bytes)
		}
		return uuid.Nil
	}
	getSecondaryID := func() uuid.UUID {
		if secondaryID.Valid {
			return uuid.UUID(secondaryID.Bytes)
		}
		return uuid.Nil
	}

	switch entity.NotificationType(notifType) {
	case entity.TypeLike:
		return entity.LikePayload{PostID: getTargetID()}
	case entity.TypeCommentLike:
		return entity.CommentLikePayload{CommentID: getTargetID()}
	case entity.TypeComment:
		return entity.CommentPayload{PostID: getTargetID(), CommentID: getSecondaryID()}
	case entity.TypeReply:
		return entity.ReplyPayload{ParentCommentID: getTargetID(), ReplyID: getSecondaryID()}
	case entity.TypeFollow:
		return entity.FollowPayload{}
	case entity.TypeRepost:
		return entity.RepostPayload{PostID: getTargetID()}
	case entity.TypeMention:
		return entity.MentionPayload{PostID: getTargetID()}
	case entity.TypeSystemAnnouncement:
		msg := ""
		if message != nil {
			msg = *message
		}
		return entity.SystemPayload{Message: msg}
	default:
		return nil
	}
}

func uuidToNullable(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
