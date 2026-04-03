package jwt

import "errors"

var (
	// ErrInvalidConfig is returned when the JWT configuration is invalid
	ErrInvalidConfig = errors.New("jwt: invalid configuration")

	// ErrInvalidToken is returned when the token is malformed or invalid
	ErrInvalidToken = errors.New("jwt: invalid token")

	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("jwt: token expired")

	// ErrTokenNotYetValid is returned when the token is not yet valid (nbf)
	ErrTokenNotYetValid = errors.New("jwt: token not yet valid")
)
