package entity

import (
	"time"

	"github.com/google/uuid"
)

// Hashtag represents a normalized hashtag stored in the database.
type Hashtag struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

// TrendingHashtag is a hashtag with its usage count in a time window.
type TrendingHashtag struct {
	Name  string
	Count int64
}
