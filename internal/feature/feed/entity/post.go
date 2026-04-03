package entity

import (
	"time"

	"github.com/google/uuid"
)

// Author holds minimal author information embedded in a feed post.
type Author struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	AvatarKey   *string
}

// Post is the feed-context view of a post.
// The app layer converts post.entity.Post → Post when crossing context boundaries.
type Post struct {
	ID           uuid.UUID
	AuthorID     uuid.UUID
	Content      string
	Visibility   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Media        []PostMedia
	LikeCount    int64
	CommentCount int64
	IsLiked      bool
	Author       *Author
}

// PostMedia is the feed-context view of a post media item.
type PostMedia struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	MediaKey  string
	MediaType string
	Position  int32
	CreatedAt time.Time
}
