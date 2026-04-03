package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims is an alias for standard JWT registered claims
// Users can extend this in their own code if they need custom claims
type Claims = jwt.RegisteredClaims

// NewClaims creates a new Claims with the given subject
func NewClaims(subject string) *Claims {
	return &Claims{
		Subject: subject,
	}
}
