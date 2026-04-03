package mailer

import (
	"context"

	"github.com/jarviisha/darkvoid/pkg/logger"
)

// NopMailer is a no-op mailer that logs emails instead of sending them.
// Used in development and testing.
type NopMailer struct{}

// Send logs the email details without actually sending.
func (m *NopMailer) Send(ctx context.Context, msg *Message) error {
	logger.Info(ctx, "nop mailer: email not sent (dev mode)",
		"to", msg.To,
		"subject", msg.Subject,
	)
	return nil
}
