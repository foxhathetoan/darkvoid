package entity

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the kind of notification.
type NotificationType string

const (
	TypeLike               NotificationType = "like"
	TypeCommentLike        NotificationType = "comment_like"
	TypeComment            NotificationType = "comment"
	TypeReply              NotificationType = "reply"
	TypeFollow             NotificationType = "follow"
	TypeRepost             NotificationType = "repost"
	TypeMention            NotificationType = "mention"
	TypeSystemAnnouncement NotificationType = "system_announcement"
)

// Actor holds minimal user info for notification display.
type Actor struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	AvatarKey   *string
}

// Payload is a sealed interface — exactly one concrete type per NotificationType.
type Payload interface {
	notificationPayload()
}

// LikePayload carries data for TypeLike.
type LikePayload struct{ PostID uuid.UUID }

func (LikePayload) notificationPayload() {}

// CommentLikePayload carries data for TypeCommentLike.
type CommentLikePayload struct{ CommentID uuid.UUID }

func (CommentLikePayload) notificationPayload() {}

// CommentPayload carries data for TypeComment.
type CommentPayload struct {
	PostID    uuid.UUID
	CommentID uuid.UUID
}

func (CommentPayload) notificationPayload() {}

// ReplyPayload carries data for TypeReply.
type ReplyPayload struct {
	ParentCommentID uuid.UUID
	ReplyID         uuid.UUID
}

func (ReplyPayload) notificationPayload() {}

// FollowPayload carries data for TypeFollow.
type FollowPayload struct{}

func (FollowPayload) notificationPayload() {}

// RepostPayload carries data for TypeRepost.
type RepostPayload struct{ PostID uuid.UUID }

func (RepostPayload) notificationPayload() {}

// MentionPayload carries data for TypeMention.
type MentionPayload struct{ PostID uuid.UUID }

func (MentionPayload) notificationPayload() {}

// SystemPayload carries data for TypeSystemAnnouncement.
type SystemPayload struct{ Message string }

func (SystemPayload) notificationPayload() {}

// Notification represents a single notification row.
type Notification struct {
	ID          uuid.UUID
	RecipientID uuid.UUID
	ActorID     uuid.UUID
	Type        NotificationType
	Payload     Payload
	GroupKey    string
	IsRead      bool
	CreatedAt   time.Time

	// Enriched fields (not stored in DB, populated at service layer)
	Actor       *Actor
	GroupCount  int64
	GroupActors []*Actor
}
