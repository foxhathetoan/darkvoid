package logger

import (
	"context"
	"log/slog"
)

// Context keys for logger
type contextKey string

const (
	loggerKey contextKey = "logger"
)

// WithLogger adds logger to context
// Performance: Reuse logger from context instead of creating new ones
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves logger from context
// Performance: Returns cached logger, avoiding allocations
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	// Return default logger if not found
	// Note: This creates a new logger - prefer setting logger in context at request start
	return New(nil)
}

// WithRequestID adds request ID to context and logger
// Performance: Creates new logger with request_id, reuse this logger in context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx)
	// Create logger with request_id once, reuse throughout request lifecycle
	newLogger := logger.With("request_id", requestID)
	return WithLogger(ctx, newLogger)
}

// WithUserID adds user ID to context and logger
// Performance: Creates new logger with user_id, reuse this logger in context
func WithUserID(ctx context.Context, userID string) context.Context {
	logger := FromContext(ctx)
	// Append user_id to existing logger (which may already have request_id)
	newLogger := logger.With("user_id", userID)
	return WithLogger(ctx, newLogger)
}

// WithFields adds multiple fields to logger in context
// Performance: Batch add fields to avoid multiple logger allocations
func WithFields(ctx context.Context, fields map[string]any) context.Context {
	if len(fields) == 0 {
		return ctx
	}

	logger := FromContext(ctx)
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	newLogger := logger.With(args...)
	return WithLogger(ctx, newLogger)
}

// Context-aware logging helpers

// Debug logs debug message using logger from context
func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// Info logs info message using logger from context
func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// Warn logs warning message using logger from context
func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// Error logs error message using logger from context
func Error(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}

// LogError logs error with context
func LogError(ctx context.Context, err error, msg string, args ...any) {
	FromContext(ctx).LogError(err, msg, args...)
}

// SetDefault sets the default global logger for slog
func SetDefault(logger *Logger) {
	slog.SetDefault(logger.Logger)
}
