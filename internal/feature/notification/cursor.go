package notification

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// MaxNotificationTime is a far-future sentinel used when no cursor is provided.
var MaxNotificationTime = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

// NotificationCursor is a composite pagination cursor (created_at, id) for notifications.
// Encoded as base64("unix_nano,notification_id").
type NotificationCursor struct {
	CreatedAt time.Time
	ID        string
}

// Encode returns the base64-encoded string representation of the cursor.
func (c *NotificationCursor) Encode() string {
	raw := fmt.Sprintf("%d,%s", c.CreatedAt.UnixNano(), c.ID)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeNotificationCursor parses a base64-encoded notification cursor string.
// Returns nil if the string is empty.
func DecodeNotificationCursor(s string) (*NotificationCursor, error) {
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
	return &NotificationCursor{CreatedAt: time.Unix(0, ns).UTC(), ID: parts[1]}, nil
}

// PgParams returns the cursor fields as pgx-compatible types for DB queries.
func (c *NotificationCursor) PgParams() (pgtype.Timestamptz, uuid.UUID, error) {
	ts := pgtype.Timestamptz{Time: c.CreatedAt, Valid: true}
	id, err := uuid.Parse(c.ID)
	return ts, id, err
}

// DefaultNotificationPgParams returns sentinel values used when no cursor is present.
func DefaultNotificationPgParams() (pgtype.Timestamptz, uuid.UUID) {
	return pgtype.Timestamptz{Time: MaxNotificationTime, Valid: true}, uuid.Max
}
