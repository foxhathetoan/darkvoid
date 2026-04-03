package entity

import (
	"time"

	"github.com/google/uuid"
)

// CommentMedia represents a media attachment on a comment.
type CommentMedia struct {
	ID        uuid.UUID
	CommentID uuid.UUID
	MediaKey  string
	MediaType string
	Position  int32
	CreatedAt time.Time
}

type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	AuthorID  uuid.UUID
	ParentID  *uuid.UUID
	Content   string
	LikeCount int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	// Enriched fields
	Author     *Author
	Media      []*CommentMedia
	IsLiked    bool
	Replies    []*Comment
	ReplyCount int64
	Mentions   []*MentionedUser
}
