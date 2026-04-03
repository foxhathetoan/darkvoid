package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/notification"
	"github.com/jarviisha/darkvoid/internal/feature/notification/broker"
	"github.com/jarviisha/darkvoid/internal/feature/notification/cache"
	"github.com/jarviisha/darkvoid/internal/feature/notification/entity"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

const (
	pageSize       = 20
	groupActorsCap = 3
)

// NotificationService handles notification business logic.
type NotificationService struct {
	repo       notifRepo
	cache      cache.NotificationCache
	userReader userReader
	broker     *broker.Broker // optional, nil = no SSE push
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(repo notifRepo, cache cache.NotificationCache, userReader userReader) *NotificationService {
	return &NotificationService{
		repo:       repo,
		cache:      cache,
		userReader: userReader,
	}
}

// WithBroker attaches an SSE broker for real-time push. Called at wire-up time.
func (s *NotificationService) WithBroker(b *broker.Broker) {
	s.broker = b
}

// --- Emitter methods (called by other bounded contexts) ---

// EmitLike creates a "like" notification. Skips self-notification.
func (s *NotificationService) EmitLike(ctx context.Context, actorID, recipientID, postID uuid.UUID) error {
	return s.emit(ctx, actorID, recipientID, entity.TypeLike, &postID, nil, fmt.Sprintf("like:%s", postID))
}

// EmitCommentLike creates a "comment_like" notification. Skips self-notification.
func (s *NotificationService) EmitCommentLike(ctx context.Context, actorID, recipientID, commentID uuid.UUID) error {
	return s.emit(ctx, actorID, recipientID, entity.TypeCommentLike, &commentID, nil, fmt.Sprintf("comment_like:%s", commentID))
}

// EmitComment creates a "comment" notification. Skips self-notification.
func (s *NotificationService) EmitComment(ctx context.Context, actorID, recipientID, postID, commentID uuid.UUID) error {
	return s.emit(ctx, actorID, recipientID, entity.TypeComment, &postID, &commentID, fmt.Sprintf("comment:%s", postID))
}

// EmitReply creates a "reply" notification. Skips self-notification.
func (s *NotificationService) EmitReply(ctx context.Context, actorID, recipientID, parentCommentID, replyID uuid.UUID) error {
	return s.emit(ctx, actorID, recipientID, entity.TypeReply, &parentCommentID, &replyID, fmt.Sprintf("reply:%s", parentCommentID))
}

// EmitFollow creates a "follow" notification. Skips self-notification.
func (s *NotificationService) EmitFollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return s.emit(ctx, followerID, followeeID, entity.TypeFollow, nil, nil, fmt.Sprintf("follow:%s", followeeID))
}

// EmitMention creates a "mention" notification. Skips self-notification.
func (s *NotificationService) EmitMention(ctx context.Context, actorID, recipientID, postID uuid.UUID) error {
	return s.emit(ctx, actorID, recipientID, entity.TypeMention, &postID, nil, fmt.Sprintf("mention:%s:%s", postID, recipientID))
}

// EmitSystemAnnouncement sends a system notification from an admin to a specific user.
// groupKey must be unique per broadcast so each announcement creates a distinct row.
func (s *NotificationService) EmitSystemAnnouncement(ctx context.Context, adminID, recipientID uuid.UUID, message, groupKey string) error {
	n, err := s.repo.CreateSystemNotification(ctx, recipientID, adminID, message, groupKey)
	if err != nil {
		logger.LogError(ctx, err, "failed to create system notification",
			"admin_id", adminID, "recipient_id", recipientID)
		return err
	}

	if err := s.cache.IncrementUnreadCount(ctx, recipientID); err != nil {
		logger.LogError(ctx, err, "failed to increment unread count cache", "recipient", recipientID)
	}

	s.publishSSE(ctx, recipientID, n)
	return nil
}

// DeleteNotification removes a notification (used for unlike/unfollow).
func (s *NotificationService) DeleteNotification(ctx context.Context, actorID uuid.UUID, groupKey string) error {
	if err := s.repo.DeleteByActorAndGroupKey(ctx, actorID, groupKey); err != nil {
		logger.LogError(ctx, err, "failed to delete notification", "actor_id", actorID, "group_key", groupKey)
		return err
	}
	return nil
}

// emit is the shared notification creation logic.
func (s *NotificationService) emit(ctx context.Context, actorID, recipientID uuid.UUID, notifType entity.NotificationType, targetID, secondaryID *uuid.UUID, groupKey string) error {
	// Never notify yourself
	if actorID == recipientID {
		return nil
	}

	n, err := s.repo.Create(ctx, recipientID, actorID, string(notifType), targetID, secondaryID, groupKey)
	if err != nil {
		logger.LogError(ctx, err, "failed to create notification",
			"type", notifType, "actor", actorID, "recipient", recipientID, "group_key", groupKey)
		return err
	}

	// Best-effort: increment cached unread count
	if err := s.cache.IncrementUnreadCount(ctx, recipientID); err != nil {
		logger.LogError(ctx, err, "failed to increment unread count cache", "recipient", recipientID)
	}

	// Best-effort: push real-time SSE event
	s.publishSSE(ctx, recipientID, n)

	return nil
}

// publishSSE pushes a real-time event to the SSE broker (fire-and-forget).
func (s *NotificationService) publishSSE(ctx context.Context, recipientID uuid.UUID, n *entity.Notification) {
	if s.broker == nil {
		return
	}

	// Enrich actor info before sending
	s.enrichActors(ctx, []*entity.Notification{n})

	payload := sseNotificationPayload{
		ID:        n.ID.String(),
		Type:      string(n.Type),
		GroupKey:  n.GroupKey,
		CreatedAt: n.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if n.Actor != nil {
		payload.Actor = &sseActorPayload{
			ID:          n.Actor.ID.String(),
			Username:    n.Actor.Username,
			DisplayName: n.Actor.DisplayName,
		}
	}
	switch p := n.Payload.(type) {
	case entity.LikePayload:
		s := p.PostID.String()
		payload.TargetID = &s
	case entity.CommentLikePayload:
		s := p.CommentID.String()
		payload.TargetID = &s
	case entity.CommentPayload:
		s := p.PostID.String()
		payload.TargetID = &s
		s2 := p.CommentID.String()
		payload.SecondaryID = &s2
	case entity.ReplyPayload:
		s := p.ParentCommentID.String()
		payload.TargetID = &s
		s2 := p.ReplyID.String()
		payload.SecondaryID = &s2
	case entity.RepostPayload:
		s := p.PostID.String()
		payload.TargetID = &s
	case entity.MentionPayload:
		s := p.PostID.String()
		payload.TargetID = &s
	case entity.SystemPayload:
		payload.Message = &p.Message
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.LogError(ctx, err, "failed to marshal SSE notification payload")
		return
	}

	s.broker.Publish(ctx, recipientID, broker.Event{
		Type: "notification",
		Data: data,
	})
}

type sseActorPayload struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type sseNotificationPayload struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Actor       *sseActorPayload `json:"actor,omitempty"`
	TargetID    *string          `json:"target_id,omitempty"`
	SecondaryID *string          `json:"secondary_id,omitempty"`
	GroupKey    string           `json:"group_key"`
	Message     *string          `json:"message,omitempty"`
	CreatedAt   string           `json:"created_at"`
}

// --- Read methods (called by handler) ---

// GetNotifications returns cursor-paginated notifications for the user, enriched with actor info.
func (s *NotificationService) GetNotifications(ctx context.Context, userID uuid.UUID, cursor *notification.NotificationCursor) ([]*entity.Notification, *notification.NotificationCursor, error) {
	var cursorTS, cursorID = notification.DefaultNotificationPgParams()

	if cursor != nil {
		var err error
		cursorTS, cursorID, err = cursor.PgParams()
		if err != nil {
			return nil, nil, errors.NewBadRequestError("invalid cursor")
		}
	}

	// Fetch one extra to detect next page
	items, err := s.repo.GetByRecipientWithCursor(ctx, userID, cursorTS, cursorID, int32(pageSize+1))
	if err != nil {
		logger.LogError(ctx, err, "failed to get notifications", "user_id", userID)
		return nil, nil, errors.NewInternalError(err)
	}

	var nextCursor *notification.NotificationCursor
	if len(items) > pageSize {
		last := items[pageSize-1]
		nextCursor = &notification.NotificationCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID.String(),
		}
		items = items[:pageSize]
	}

	s.enrichActors(ctx, items)
	s.enrichGroups(ctx, userID, items)

	return items, nextCursor, nil
}

// GetUnreadCount returns the unread notification count (cache-first, fallback to DB).
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.cache.GetUnreadCount(ctx, userID)
	if err != nil {
		logger.LogError(ctx, err, "unread count cache read error", "user_id", userID)
	} else if count >= 0 {
		return count, nil
	}

	// Cache miss → query DB
	count, err = s.repo.CountUnread(ctx, userID)
	if err != nil {
		logger.LogError(ctx, err, "failed to count unread notifications", "user_id", userID)
		return 0, errors.NewInternalError(err)
	}

	if cacheErr := s.cache.SetUnreadCount(ctx, userID, count); cacheErr != nil {
		logger.LogError(ctx, cacheErr, "unread count cache write error", "user_id", userID)
	}

	return count, nil
}

// MarkAsRead marks a single notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error {
	if err := s.repo.MarkAsRead(ctx, notificationID, userID); err != nil {
		logger.LogError(ctx, err, "failed to mark notification as read", "notification_id", notificationID)
		return errors.NewInternalError(err)
	}
	if err := s.cache.InvalidateUnreadCount(ctx, userID); err != nil {
		logger.LogError(ctx, err, "failed to invalidate unread count cache", "user_id", userID)
	}
	return nil
}

// MarkAllAsRead marks all notifications as read for the user.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.MarkAllAsRead(ctx, userID); err != nil {
		logger.LogError(ctx, err, "failed to mark all notifications as read", "user_id", userID)
		return errors.NewInternalError(err)
	}
	if err := s.cache.InvalidateUnreadCount(ctx, userID); err != nil {
		logger.LogError(ctx, err, "failed to invalidate unread count cache", "user_id", userID)
	}
	return nil
}

// --- Enrichment helpers ---

// enrichActors batch-fetches actor info and sets Notification.Actor.
func (s *NotificationService) enrichActors(ctx context.Context, items []*entity.Notification) {
	if len(items) == 0 {
		return
	}

	seen := make(map[uuid.UUID]bool, len(items))
	ids := make([]uuid.UUID, 0, len(items))
	for _, n := range items {
		if !seen[n.ActorID] {
			seen[n.ActorID] = true
			ids = append(ids, n.ActorID)
		}
	}

	authors, err := s.userReader.GetAuthorsByIDs(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich notification actors")
		return
	}

	for _, n := range items {
		if a, ok := authors[n.ActorID]; ok {
			n.Actor = a
		}
	}
}

// enrichGroups fetches group actor counts for distinct group_keys in the result set.
func (s *NotificationService) enrichGroups(ctx context.Context, userID uuid.UUID, items []*entity.Notification) {
	if len(items) == 0 {
		return
	}

	// Collect unique group keys
	seen := make(map[string]bool)
	for _, n := range items {
		seen[n.GroupKey] = true
	}

	// Fetch group info per unique group_key
	groupInfo := make(map[string]struct {
		actorIDs []uuid.UUID
		total    int64
	})
	for gk := range seen {
		actorIDs, total, err := s.repo.GetGroupActors(ctx, userID, gk, groupActorsCap)
		if err != nil {
			logger.LogError(ctx, err, "failed to get group actors", "group_key", gk)
			continue
		}
		groupInfo[gk] = struct {
			actorIDs []uuid.UUID
			total    int64
		}{actorIDs: actorIDs, total: total}
	}

	// Batch-fetch all unique actor IDs from groups
	allActorIDs := make(map[uuid.UUID]bool)
	for _, gi := range groupInfo {
		for _, id := range gi.actorIDs {
			allActorIDs[id] = true
		}
	}
	ids := make([]uuid.UUID, 0, len(allActorIDs))
	for id := range allActorIDs {
		ids = append(ids, id)
	}

	var authors map[uuid.UUID]*entity.Actor
	if len(ids) > 0 {
		var err error
		authors, err = s.userReader.GetAuthorsByIDs(ctx, ids)
		if err != nil {
			logger.LogError(ctx, err, "failed to enrich group actors")
			authors = make(map[uuid.UUID]*entity.Actor)
		}
	}

	// Assign group data to each notification
	for _, n := range items {
		gi, ok := groupInfo[n.GroupKey]
		if !ok {
			continue
		}
		n.GroupCount = gi.total
		groupActors := make([]*entity.Actor, 0, len(gi.actorIDs))
		for _, id := range gi.actorIDs {
			if a, ok := authors[id]; ok {
				groupActors = append(groupActors, a)
			}
		}
		n.GroupActors = groupActors
	}
}
