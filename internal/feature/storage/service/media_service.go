package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	featurestorage "github.com/jarviisha/darkvoid/internal/feature/storage"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// MediaType classifies the uploaded file.
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

// UploadResult is returned after a successful upload.
type UploadResult struct {
	Key       string    `json:"key"`
	URL       string    `json:"url"`
	MediaType MediaType `json:"media_type"`
}

// allowed MIME types and their corresponding MediaType + extension
var allowedMIME = map[string]struct {
	mediaType MediaType
	ext       string
}{
	"image/jpeg": {MediaTypeImage, ".jpg"},
	"image/png":  {MediaTypeImage, ".png"},
	"image/webp": {MediaTypeImage, ".webp"},
	"image/gif":  {MediaTypeImage, ".gif"},
	"video/mp4":  {MediaTypeVideo, ".mp4"},
	"video/webm": {MediaTypeVideo, ".webm"},
}

const (
	maxImageSize int64 = 10 << 20  // 10 MB
	maxVideoSize int64 = 100 << 20 // 100 MB
)

// MediaService handles validation and upload of media files.
type MediaService struct {
	storage storage.Storage
}

// NewMediaService creates a new MediaService.
func NewMediaService(s storage.Storage) *MediaService {
	return &MediaService{storage: s}
}

// Upload validates and uploads a file to storage under the "media/" prefix.
// contentType must be an explicit MIME type (from Content-Type header or sniffed).
// filename is used only to derive the extension as fallback when MIME lookup gives a generic ext.
func (s *MediaService) Upload(ctx context.Context, r io.Reader, size int64, contentType, filename string) (*UploadResult, error) {
	// Normalize: strip parameters like "image/jpeg; charset=utf-8"
	mime := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	meta, ok := allowedMIME[mime]
	if !ok {
		logger.Warn(ctx, "unsupported media type", "content_type", contentType)
		return nil, featurestorage.ErrUnsupportedType
	}

	maxSize := maxImageSize
	if meta.mediaType == MediaTypeVideo {
		maxSize = maxVideoSize
	}
	if size > maxSize {
		logger.Warn(ctx, "file too large", "size", size, "max", maxSize)
		return nil, featurestorage.ErrFileTooLarge
	}

	// Prefer ext from MIME; fall back to filename ext
	ext := meta.ext
	if filenameExt := strings.ToLower(filepath.Ext(filename)); filenameExt != "" && ext == "" {
		ext = filenameExt
	}

	key := fmt.Sprintf("media/%s%s", uuid.New().String(), ext)

	if err := s.storage.Put(ctx, key, r, size, mime); err != nil {
		logger.LogError(ctx, err, "failed to upload media", "key", key)
		return nil, errors.NewInternalError(err)
	}

	logger.Info(ctx, "media uploaded", "key", key, "media_type", meta.mediaType)
	return &UploadResult{
		Key:       key,
		URL:       s.storage.URL(key),
		MediaType: meta.mediaType,
	}, nil
}
