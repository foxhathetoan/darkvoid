package post

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UserPostCursor is a composite pagination cursor (created_at, post_id) for user post listing.
// Encoded as base64("unix_nano,post_id").
type UserPostCursor struct {
	CreatedAt time.Time
	PostID    string
}

// Encode returns the base64-encoded string representation of the cursor.
func (c *UserPostCursor) Encode() string {
	raw := fmt.Sprintf("%d,%s", c.CreatedAt.UnixNano(), c.PostID)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeUserPostCursor parses a base64-encoded user post cursor string.
// Returns nil if the string is empty.
func DecodeUserPostCursor(s string) (*UserPostCursor, error) {
	if s == "" {
		return nil, nil //nolint:nilnil // empty string means no cursor
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	parts := strings.SplitN(string(b), ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}
	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return &UserPostCursor{CreatedAt: time.Unix(0, ns).UTC(), PostID: parts[1]}, nil
}

// MaxUserPostTime is a sentinel used when no cursor is present — fetches from the far future so all posts are included.
var MaxUserPostTime = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
