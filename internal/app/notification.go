package app

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarviisha/darkvoid/internal/app/middleware"
	"github.com/jarviisha/darkvoid/internal/feature/notification/broker"
	notifcache "github.com/jarviisha/darkvoid/internal/feature/notification/cache"
	notifhandler "github.com/jarviisha/darkvoid/internal/feature/notification/handler"
	"github.com/jarviisha/darkvoid/internal/feature/notification/repository"
	notifservice "github.com/jarviisha/darkvoid/internal/feature/notification/service"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// NotificationContext represents the Notification bounded context.
type NotificationContext struct {
	NotifService *notifservice.NotificationService
	NotifHandler *notifhandler.NotificationHandler
	Broker       *broker.Broker // nil when Redis is disabled
}

// SetupNotificationContext initializes the Notification context with all required dependencies.
// redisClient may be nil — in that case a no-op cache is used and SSE operates in-memory only.
// After creating the service, it wires notification emitters into the producing services
// (LikeService, CommentService, FollowService) using the deferred wiring pattern.
func SetupNotificationContext(pool *pgxpool.Pool, store storage.Storage, userCtx *UserContext, postCtx *PostContext, followService *service.FollowService, redisClient *pkgredis.Client) *NotificationContext {
	notifRepo := repository.NewNotificationRepository(pool)

	// Build cache: Redis when available, no-op otherwise
	var nc notifcache.NotificationCache
	if redisClient != nil {
		nc = notifcache.NewRedisNotificationCache(redisClient)
	} else {
		nc = notifcache.NewNopNotificationCache()
	}

	// User reader adapter for enriching actor info
	ur := &notifUserReader{userRepo: userCtx.UserRepo}

	// Build SSE broker
	b := broker.NewBroker(redisClient)

	notifSvc := notifservice.NewNotificationService(notifRepo, nc, ur)
	notifSvc.WithBroker(b)

	notifHdlr := notifhandler.NewNotificationHandler(notifSvc, store)
	notifHdlr.WithBroker(b)

	// Wire notification emitters into producing services (deferred wiring)
	postCtx.PostService.WithNotificationEmitter(notifSvc)
	postCtx.LikeService.WithNotificationEmitter(notifSvc)
	postCtx.CommentService.WithNotificationEmitter(notifSvc)
	postCtx.CommentLikeService.WithNotificationEmitter(notifSvc)
	followService.WithNotificationEmitter(notifSvc)

	return &NotificationContext{
		NotifService: notifSvc,
		NotifHandler: notifHdlr,
		Broker:       b,
	}
}

// StartBroker starts the Redis Pub/Sub subscriber in a background goroutine.
// Should be called after the application is fully initialized.
// No-op when Redis is disabled (broker operates in-memory only).
func (ctx *NotificationContext) StartBroker(appCtx context.Context) {
	if ctx.Broker != nil {
		go ctx.Broker.StartRedisSubscriber(appCtx)
	}
}

// RegisterRoutes registers all routes for the Notification context.
func (ctx *NotificationContext) RegisterRoutes(r chi.Router, auth middleware.AuthMiddleware) {
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)
		r.Get("/notifications", ctx.NotifHandler.GetNotifications)
		r.Get("/notifications/unread-count", ctx.NotifHandler.GetUnreadCount)
		r.Post("/notifications/{notificationID}/read", ctx.NotifHandler.MarkAsRead)
		r.Post("/notifications/read-all", ctx.NotifHandler.MarkAllAsRead)
	})
}

// RegisterSSERoute registers the SSE stream endpoint outside the normal timeout middleware.
// This must be called separately because SSE connections are long-lived and must not
// be subject to the global request timeout.
func (ctx *NotificationContext) RegisterSSERoute(r chi.Router, auth middleware.AuthMiddleware) {
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)
		r.Get("/notifications/stream", ctx.NotifHandler.Stream)
	})
}
