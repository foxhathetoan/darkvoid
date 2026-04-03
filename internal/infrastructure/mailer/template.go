package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// Templates holds parsed email templates.
type Templates struct {
	welcome       *template.Template
	verifyEmail   *template.Template
	resetPassword *template.Template
}

// LoadTemplates parses all embedded email templates.
func LoadTemplates() (*Templates, error) {
	parse := func(name string) (*template.Template, error) {
		t, err := template.ParseFS(templateFS, "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		return t, nil
	}

	welcome, err := parse("welcome.html")
	if err != nil {
		return nil, err
	}
	verify, err := parse("verify_email.html")
	if err != nil {
		return nil, err
	}
	reset, err := parse("reset_password.html")
	if err != nil {
		return nil, err
	}

	return &Templates{
		welcome:       welcome,
		verifyEmail:   verify,
		resetPassword: reset,
	}, nil
}

// WelcomeData holds template data for the welcome email.
type WelcomeData struct {
	Username string
}

// VerifyEmailData holds template data for the verification email.
type VerifyEmailData struct {
	Username  string
	VerifyURL string
	ExpiresIn string
}

// ResetPasswordData holds template data for the password reset email.
type ResetPasswordData struct {
	Username  string
	ResetURL  string
	ExpiresIn string
}

// RenderWelcome renders the welcome email template.
func (t *Templates) RenderWelcome(data WelcomeData) (string, error) {
	return render(t.welcome, data)
}

// RenderVerifyEmail renders the email verification template.
func (t *Templates) RenderVerifyEmail(data VerifyEmailData) (string, error) {
	return render(t.verifyEmail, data)
}

// RenderResetPassword renders the password reset template.
func (t *Templates) RenderResetPassword(data ResetPasswordData) (string, error) {
	return render(t.resetPassword, data)
}

func render(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return buf.String(), nil
}
