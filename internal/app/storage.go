package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/jarviisha/darkvoid/internal/app/middleware"
	feathandler "github.com/jarviisha/darkvoid/internal/feature/storage/handler"
	featsvc "github.com/jarviisha/darkvoid/internal/feature/storage/service"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// StorageContext represents the Storage bounded context.
type StorageContext struct {
	MediaService *featsvc.MediaService
	MediaHandler *feathandler.MediaHandler
}

// SetupStorageContext initializes the Storage context.
func SetupStorageContext(store storage.Storage) *StorageContext {
	mediaService := featsvc.NewMediaService(store)
	mediaHandler := feathandler.NewMediaHandler(mediaService)

	return &StorageContext{
		MediaService: mediaService,
		MediaHandler: mediaHandler,
	}
}

// RegisterRoutes registers all routes for the Storage context.
func (ctx *StorageContext) RegisterRoutes(r chi.Router, auth middleware.AuthMiddleware) {
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)
		r.Post("/media/upload", ctx.MediaHandler.Upload)
	})
}
