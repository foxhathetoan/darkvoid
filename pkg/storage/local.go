package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// localStorage stores files on the local filesystem.
// Files are served via a static file server mounted at BaseURL.
type localStorage struct {
	dir     string // absolute path to upload directory, e.g. "/app/uploads"
	baseURL string // public base URL, e.g. "http://localhost:8080/static"
}

// NewLocal returns a Storage backed by the local filesystem.
//
// dir is the directory where files will be written.
// baseURL is the public-facing base URL used to build download URLs.
//
// Example:
//
//	NewLocal("./uploads", "http://localhost:8080/static")
//	// key "avatars/abc.jpg" → URL "http://localhost:8080/static/avatars/abc.jpg"
func NewLocal(dir, baseURL string) (Storage, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("storage/local: resolve dir %q: %w", dir, err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("storage/local: create dir %q: %w", abs, err)
	}
	return &localStorage{
		dir:     abs,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

// Put writes r to dir/key, creating parent directories as needed.
// size and contentType are accepted for interface compatibility but not used by the local provider.
func (l *localStorage) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	dest := filepath.Join(l.dir, filepath.FromSlash(key))

	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return fmt.Errorf("storage/local: mkdir for key %q: %w", key, err)
	}

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("storage/local: open %q: %w", dest, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage/local: write %q: %w", dest, err)
	}
	return nil
}

// Delete removes the file at dir/key.
// Returns nil if the file does not exist (idempotent).
func (l *localStorage) Delete(_ context.Context, key string) error {
	dest := filepath.Join(l.dir, filepath.FromSlash(key))
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage/local: delete %q: %w", dest, err)
	}
	return nil
}

// URL builds the public URL for a given key.
func (l *localStorage) URL(key string) string {
	return fmt.Sprintf("%s/%s", l.baseURL, key)
}
