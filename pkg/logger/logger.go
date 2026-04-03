package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Format      string // json, text
	Output      io.Writer
	AddSource   bool // Add source file & line number
	Service     string
	Version     string
	Environment string
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:       "info",
		Format:      "json",
		Output:      os.Stdout,
		AddSource:   false,
		Service:     "darkvoid",
		Environment: "development",
	}
}

// New creates a new logger instance
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse level
	level := parseLevel(cfg.Level)

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	// Create handler based on format
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	// Create logger with default fields
	logger := slog.New(handler)

	// Add default attributes
	if cfg.Service != "" {
		logger = logger.With("service", cfg.Service)
	}
	if cfg.Version != "" {
		logger = logger.With("version", cfg.Version)
	}
	if cfg.Environment != "" {
		logger = logger.With("env", cfg.Environment)
	}

	return &Logger{Logger: logger}
}

// parseLevel converts string level to slog.Level
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values
	args := []any{}

	// Add request ID if available (safe type assertion)
	if reqID := ctx.Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			args = append(args, "request_id", id)
		}
	}

	// Add user ID if available (safe type assertion)
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			args = append(args, "user_id", id)
		}
	}

	if len(args) == 0 {
		return l
	}

	return &Logger{Logger: l.Logger.With(args...)}
}

// With returns a logger with additional attributes
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// WithGroup returns a logger with grouped attributes
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{Logger: l.Logger.WithGroup(name)}
}

// Helper methods for common logging patterns

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

// Helper methods for structured logging

// LogRequest logs HTTP request details
func (l *Logger) LogRequest(method, path string, statusCode int, duration float64, attrs ...any) {
	baseAttrs := make([]any, 0, 8+len(attrs))
	baseAttrs = append(baseAttrs,
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration,
	)
	allAttrs := append(baseAttrs, attrs...)
	l.Info("http request", allAttrs...)
}

// LogError logs error with details
func (l *Logger) LogError(err error, msg string, attrs ...any) {
	if err == nil {
		return
	}

	// Build attributes
	baseAttrs := make([]any, 0, 6+len(attrs))
	baseAttrs = append(baseAttrs,
		"error", err.Error(),
		"error_type", fmt.Sprintf("%T", err),
	)

	// Add wrapped error chain if available
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		baseAttrs = append(baseAttrs, "cause", unwrapped.Error())
	}

	allAttrs := append(baseAttrs, attrs...)
	l.Error(msg, allAttrs...)
}

// LogDB logs database operation
func (l *Logger) LogDB(operation, table string, duration float64, attrs ...any) {
	baseAttrs := make([]any, 0, 6+len(attrs))
	baseAttrs = append(baseAttrs,
		"operation", operation,
		"table", table,
		"duration_ms", duration,
	)
	allAttrs := append(baseAttrs, attrs...)
	l.Debug("database operation", allAttrs...)
}

// LogAuth logs authentication events
func (l *Logger) LogAuth(event, userID string, success bool, attrs ...any) {
	baseAttrs := make([]any, 0, 6+len(attrs))
	baseAttrs = append(baseAttrs,
		"event", event,
		"user_id", userID,
		"success", success,
	)
	allAttrs := append(baseAttrs, attrs...)
	l.Info("auth event", allAttrs...)
}
