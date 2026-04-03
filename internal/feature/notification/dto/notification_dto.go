package dto

import (
	"github.com/jarviisha/darkvoid/internal/feature/notification/entity"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// ActorResponse holds minimal author info for notification display.
type ActorResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// NotificationResponse is the outgoing representation of a single notification.
type NotificationResponse struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Actor       *ActorResponse  `json:"actor,omitempty"`
	GroupActors []ActorResponse `json:"group_actors,omitempty"`
	GroupCount  int64           `json:"group_count"`
	TargetID    *string         `json:"target_id,omitempty"`
	SecondaryID *string         `json:"secondary_id,omitempty"`
	Message     *string         `json:"message,omitempty"`
	IsRead      bool            `json:"is_read"`
	CreatedAt   string          `json:"created_at"`
}

// NotificationListResponse is the response for GET /notifications.
type NotificationListResponse struct {
	Data       []NotificationResponse `json:"data"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

// UnreadCountResponse is the response for GET /notifications/unread-count.
type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

// ToNotificationResponse converts entity to DTO.
func ToNotificationResponse(n *entity.Notification, store storage.Storage) NotificationResponse {
	resp := NotificationResponse{
		ID:         n.ID.String(),
		Type:       string(n.Type),
		GroupCount: n.GroupCount,
		IsRead:     n.IsRead,
		CreatedAt:  n.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	switch p := n.Payload.(type) {
	case entity.LikePayload:
		s := p.PostID.String()
		resp.TargetID = &s
	case entity.CommentLikePayload:
		s := p.CommentID.String()
		resp.TargetID = &s
	case entity.CommentPayload:
		s := p.PostID.String()
		resp.TargetID = &s
		s2 := p.CommentID.String()
		resp.SecondaryID = &s2
	case entity.ReplyPayload:
		s := p.ParentCommentID.String()
		resp.TargetID = &s
		s2 := p.ReplyID.String()
		resp.SecondaryID = &s2
	case entity.RepostPayload:
		s := p.PostID.String()
		resp.TargetID = &s
	case entity.MentionPayload:
		s := p.PostID.String()
		resp.TargetID = &s
	case entity.SystemPayload:
		resp.Message = &p.Message
	}

	if n.Actor != nil {
		resp.Actor = toActorResponse(n.Actor, store)
	}

	if len(n.GroupActors) > 0 {
		resp.GroupActors = make([]ActorResponse, len(n.GroupActors))
		for i, a := range n.GroupActors {
			resp.GroupActors[i] = *toActorResponse(a, store)
		}
	}

	return resp
}

func toActorResponse(a *entity.Actor, store storage.Storage) *ActorResponse {
	resp := &ActorResponse{
		ID:          a.ID.String(),
		Username:    a.Username,
		DisplayName: a.DisplayName,
	}
	if a.AvatarKey != nil {
		url := store.URL(*a.AvatarKey)
		resp.AvatarURL = &url
	}
	return resp
}
