package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Config mirrors config.StorageConfig to avoid import cycle.
type Config struct {
	Provider string
	BaseURL  string
	LocalDir string
}

// New creates a Storage from config, selecting the appropriate provider.
// Supported providers: "local". Falls back to nop for unknown providers.
func New(cfg Config) (Storage, error) {
	switch cfg.Provider {
	case "local":
		return NewLocal(cfg.LocalDir, cfg.BaseURL)
	default:
		// Unknown or empty provider — use nop (URL-only, no real I/O)
		return NewNop(cfg.BaseURL), nil
	}
}

// Storage defines the interface for file storage operations.
// Implementations are provider-agnostic (local disk, S3, GCS, MinIO, etc.)
type Storage interface {
	// Put uploads a file and stores it at the given key.
	// key is a relative path like "avatars/abc123.jpg"
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Delete removes the file at the given key.
	// Returns nil if the file does not exist (idempotent).
	Delete(ctx context.Context, key string) error

	// URL builds the public-accessible URL for a given key.
	// This is called at response time, never stored in the database.
	URL(key string) string
}

// nopStorage is a no-op placeholder used until a real provider is configured.
// Put and Delete succeed silently; URL builds a URL from baseURL + key.
type nopStorage struct {
	baseURL string
}

// NewNop returns a Storage implementation that performs no real I/O.
// It is intended as a placeholder until a concrete provider (local, S3, etc.) is wired in.
func NewNop(baseURL string) Storage {
	return &nopStorage{baseURL: strings.TrimRight(baseURL, "/")}
}

func (n *nopStorage) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}

func (n *nopStorage) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *nopStorage) URL(key string) string {
	return fmt.Sprintf("%s/%s", n.baseURL, key)
}
