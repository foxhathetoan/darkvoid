package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// RateLimitByIP returns a middleware that limits requests per IP address.
// limit is the maximum number of requests allowed within the window duration.
// Responds with 429 Too Many Requests when the limit is exceeded.
func RateLimitByIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.Limit(
		limit,
		window,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			errors.WriteErrorResponse(w, errors.New("RATE_LIMIT_EXCEEDED", "too many requests, please slow down", http.StatusTooManyRequests))
		}),
	)
}
