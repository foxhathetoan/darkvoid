package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/jwt"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// AuthMiddleware bundles the two auth middleware variants so that bounded contexts
// can apply authentication without knowing about the JWT implementation.
type AuthMiddleware struct {
	Required func(http.Handler) http.Handler // 401 when token is missing/invalid
	Optional func(http.Handler) http.Handler // continues without auth on failure
}

// NewAuthMiddleware creates an AuthMiddleware from a JWT service.
func NewAuthMiddleware(jwtService *jwt.Service) AuthMiddleware {
	return AuthMiddleware{
		Required: Auth(jwtService),
		Optional: OptionalAuth(jwtService),
	}
}

// extractToken extracts a JWT from the request.
// Priority: Authorization header → ?token= query parameter.
func extractToken(r *http.Request) string {
	// 1. Check Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if token := strings.TrimPrefix(authHeader, "Bearer "); token != authHeader {
			return token
		}
	}
	// 2. Fallback to query parameter (used by SSE/EventSource)
	return r.URL.Query().Get("token")
}

// Auth creates an authentication middleware that validates JWT tokens.
// Accepts token from Authorization header or ?token= query parameter.
func Auth(jwtService *jwt.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract token from header or query param
			tokenString := extractToken(r)
			if tokenString == "" {
				logger.Warn(ctx, "missing authorization token")
				errors.WriteErrorResponse(w, errors.NewUnauthorizedError("missing authorization token"))
				return
			}

			// Validate token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				logger.Warn(ctx, "token validation failed", "error", err)

				switch err {
				case jwt.ErrExpiredToken:
					errors.WriteErrorResponse(w, errors.NewUnauthorizedError("token expired"))
				case jwt.ErrInvalidToken:
					errors.WriteErrorResponse(w, errors.NewUnauthorizedError("invalid token"))
				case jwt.ErrTokenNotYetValid:
					errors.WriteErrorResponse(w, errors.NewUnauthorizedError("token not yet valid"))
				default:
					errors.WriteErrorResponse(w, errors.ErrUnauthorized)
				}
				return
			}

			// Validate subject (user ID) exists
			if claims.Subject == "" {
				logger.Warn(ctx, "token missing subject")
				errors.WriteErrorResponse(w, errors.NewUnauthorizedError("invalid token claims"))
				return
			}

			// Parse user ID from subject
			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				logger.Warn(ctx, "invalid user ID format in token", "subject", claims.Subject, "error", err)
				errors.WriteErrorResponse(w, errors.NewUnauthorizedError("invalid token claims"))
				return
			}

			// Store user ID in context
			ctx = httputil.WithUserID(ctx, userID)

			logger.Info(ctx, "user authenticated", "user_id", userID)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth creates an optional authentication middleware.
// If token is present and valid, user info is stored in context.
// If token is missing or invalid, request continues without authentication.
// Accepts token from Authorization header or ?token= query parameter.
func OptionalAuth(jwtService *jwt.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tokenString := extractToken(r)
			if tokenString == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Validate token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				// Invalid token, continue without authentication
				logger.Debug(ctx, "optional auth: token validation failed", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// Store user ID in context if valid
			if claims.Subject != "" {
				userID, err := uuid.Parse(claims.Subject)
				if err == nil {
					ctx = httputil.WithUserID(ctx, userID)
					logger.Debug(ctx, "optional auth: user authenticated", "user_id", userID)
				} else {
					logger.Debug(ctx, "optional auth: invalid user ID format", "subject", claims.Subject)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
