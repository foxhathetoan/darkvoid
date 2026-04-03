package mailer

import (
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/jarviisha/darkvoid/pkg/logger"
)

// SMTPMailer sends emails via SMTP.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPMailer creates a new SMTP mailer from the given config.
func NewSMTPMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
	}
}

// Send sends an email via SMTP with PLAIN auth and TLS on port 587.
func (m *SMTPMailer) Send(ctx context.Context, msg *Message) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	// Envelope sender must be a bare email address (RFC 5321).
	// m.from may contain a display name like "DarkVoid <noreply@darkvoid.app>".
	envelopeFrom := extractEmail(m.from)

	body := buildMIME(m.from, msg)

	if err := smtp.SendMail(addr, auth, envelopeFrom, msg.To, []byte(body)); err != nil {
		logger.Error(ctx, "failed to send email", "to", msg.To, "subject", msg.Subject, "error", err)
		return fmt.Errorf("send email to %s: %w", strings.Join(msg.To, ","), err)
	}

	logger.Info(ctx, "email sent", "to", msg.To, "subject", msg.Subject)
	return nil
}

// extractEmail parses a "Name <email>" string and returns just the email address.
// If parsing fails, it returns the input as-is.
func extractEmail(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return from
	}
	return addr.Address
}

// buildMIME constructs a multipart MIME email with HTML and plain text parts.
func buildMIME(from string, msg *Message) string {
	boundary := "==DarkVoidBoundary=="
	var b strings.Builder

	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ",") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
	b.WriteString("\r\n")

	if msg.Text != "" {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.Text + "\r\n")
	}

	if msg.HTML != "" {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.HTML + "\r\n")
	}

	b.WriteString("--" + boundary + "--\r\n")
	return b.String()
}
