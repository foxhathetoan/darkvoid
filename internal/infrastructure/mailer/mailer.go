package mailer

import "context"

// Mailer sends emails. Implementations include SMTP and a no-op stub for testing/development.
type Mailer interface {
	Send(ctx context.Context, msg *Message) error
}

// Message represents an email to be sent.
type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string // plain text fallback
}

// Config holds mailer configuration.
type Config struct {
	// Provider selects the mailer backend: "smtp" or "nop"
	Provider string

	// SMTP settings
	Host     string
	Port     int
	Username string
	Password string
	From     string

	// BaseURL is the application URL used to build links in emails (e.g. verification links).
	BaseURL string
}

// New creates a Mailer based on the provider specified in cfg.
// Falls back to NopMailer for unknown providers.
func New(cfg Config) Mailer {
	switch cfg.Provider {
	case "smtp":
		return NewSMTPMailer(cfg)
	default:
		return &NopMailer{}
	}
}
