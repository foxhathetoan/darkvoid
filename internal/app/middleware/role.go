package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// RoleChecker checks whether a user holds any of the specified roles.
// Implemented by AdminService and wired at app init time.
type RoleChecker interface {
	UserHasAnyRole(ctx context.Context, userID uuid.UUID, roles []string) (bool, error)
}

// RequireRole returns a middleware that rejects requests from users that do not
// hold at least one of the given roles. Must be applied after AuthRequired.
func RequireRole(checker RoleChecker, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			userID := httputil.GetUserID(ctx)
			if userID == nil {
				errors.WriteErrorResponse(w, errors.NewUnauthorizedError("missing authorization token"))
				return
			}

			ok, err := checker.UserHasAnyRole(ctx, *userID, roles)
			if err != nil {
				logger.LogError(ctx, err, "role check failed", "user_id", *userID)
				errors.WriteErrorResponse(w, errors.NewInternalError(err))
				return
			}
			if !ok {
				logger.Warn(ctx, "access denied: insufficient role", "user_id", *userID, "required_roles", roles)
				errors.WriteErrorResponse(w, errors.NewForbiddenError("insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
