package handler

import (
	"net/http"

	"github.com/jarviisha/darkvoid/internal/feature/storage/service"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

const maxFormMemory = 32 << 20 // 32 MB multipart buffer

// MediaHandler handles media upload requests.
type MediaHandler struct {
	mediaService *service.MediaService
}

// NewMediaHandler creates a new MediaHandler.
func NewMediaHandler(mediaService *service.MediaService) *MediaHandler {
	return &MediaHandler{mediaService: mediaService}
}

// Upload godoc
//
//	@Summary		Upload a media file
//	@Description	Upload an image or video file. Returns the storage key and public URL.
//	@Description	Supported types: image/jpeg, image/png, image/webp, image/gif, video/mp4, video/webm
//	@Description	Max size: 10 MB for images, 100 MB for videos.
//	@Tags			media
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"File to upload"
//	@Success		200		{object}	UploadResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Missing file"
//	@Failure		401		{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		413		{object}	errors.ErrorResponse	"File too large"
//	@Failure		415		{object}	errors.ErrorResponse	"Unsupported media type"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				uploadMedia
//	@Router			/media/upload [post]
//	@Security		BearerAuth
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, maxFormMemory)
	if err := r.ParseMultipartForm(maxFormMemory); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid or missing multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("field 'file' is required"))
		return
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback: sniff from file bytes (net/http will read at most 512 bytes)
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		contentType = http.DetectContentType(buf[:n])
		// Seek back — multipart.File implements io.ReadSeeker
		if seeker, ok := file.(interface {
			Seek(int64, int) (int64, error)
		}); ok {
			_, _ = seeker.Seek(0, 0)
		}
	}

	result, err := h.mediaService.Upload(ctx, file, header.Size, contentType, header.Filename)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "upload successful", "key", result.Key, "media_type", result.MediaType)
	httputil.WriteJSON(w, http.StatusOK, UploadResponse{
		Key:       result.Key,
		URL:       result.URL,
		MediaType: string(result.MediaType),
	})
}

// UploadResponse is the JSON body returned after a successful upload.
type UploadResponse struct {
	Key       string `json:"key" example:"media/550e8400-e29b-41d4-a716-446655440000.jpg"`
	URL       string `json:"url" example:"http://localhost:8080/static/media/550e8400.jpg"`
	MediaType string `json:"media_type" example:"image"`
}
