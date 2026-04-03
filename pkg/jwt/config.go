package jwt

import "time"

// Config holds JWT configuration
type Config struct {
	// Secret key for signing tokens
	Secret []byte

	// Issuer identifies the principal that issued the JWT
	Issuer string

	// Expiry is the duration for access token validity
	Expiry time.Duration
}

// DefaultConfig returns a default JWT configuration
// Note: Secret must be set before use, as it cannot have a sensible default
func DefaultConfig() Config {
	return Config{
		Issuer: "darkvoid",
		Expiry: 15 * time.Minute,
		// Secret must be set by caller
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if len(c.Secret) == 0 {
		return ErrInvalidConfig
	}
	if c.Expiry <= 0 {
		return ErrInvalidConfig
	}
	return nil
}
