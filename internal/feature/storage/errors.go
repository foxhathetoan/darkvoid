package storage

import (
	"net/http"

	"github.com/jarviisha/darkvoid/pkg/errors"
)

var (
	ErrFileTooLarge    = errors.New("FILE_TOO_LARGE", "file exceeds maximum allowed size", http.StatusRequestEntityTooLarge)
	ErrUnsupportedType = errors.New("UNSUPPORTED_MEDIA_TYPE", "file type is not supported", http.StatusUnsupportedMediaType)
)
