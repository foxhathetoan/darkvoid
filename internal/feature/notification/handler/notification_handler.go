package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/notification"
	"github.com/jarviisha/darkvoid/internal/feature/notification/dto"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// NotificationHandler handles HTTP requests for notifications.
type NotificationHandler struct {
	notifService notifService
	store        storage.Storage
	broker       sseBroker // optional, nil = SSE disabled
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(notifService notifService, store storage.Storage) *NotificationHandler {
	return &NotificationHandler{notifService: notifService, store: store}
}

// WithBroker attaches an SSE broker. Called at wire-up time.
func (h *NotificationHandler) WithBroker(b sseBroker) {
	h.broker = b
}

// GetNotifications godoc
//
//	@Summary		Get notifications
//	@Description	Get a cursor-paginated list of notifications for the authenticated user
//	@Tags			notifications
//	@Produce		json
//	@Param			cursor	query		string	false	"Pagination cursor from previous response"
//	@Success		200		{object}	dto.NotificationListResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getNotifications
//	@Router			/notifications [get]
//	@Security		BearerAuth
func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	cursor, err := notification.DecodeNotificationCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid cursor"))
		return
	}

	items, nextCursor, err := h.notifService.GetNotifications(ctx, *userID, cursor)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	resp := dto.NotificationListResponse{
		Data: make([]dto.NotificationResponse, len(items)),
	}
	for i, n := range items {
		resp.Data[i] = dto.ToNotificationResponse(n, h.store)
	}
	if nextCursor != nil {
		resp.NextCursor = nextCursor.Encode()
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// GetUnreadCount godoc
//
//	@Summary		Get unread notification count
//	@Description	Get the number of unread notifications for the authenticated user
//	@Tags			notifications
//	@Produce		json
//	@Success		200	{object}	dto.UnreadCountResponse
//	@Failure		401	{object}	errors.ErrorResponse
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				getUnreadCount
//	@Router			/notifications/unread-count [get]
//	@Security		BearerAuth
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	count, err := h.notifService.GetUnreadCount(ctx, *userID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.UnreadCountResponse{UnreadCount: count})
}

// MarkAsRead godoc
//
//	@Summary		Mark notification as read
//	@Description	Mark a single notification as read
//	@Tags			notifications
//	@Produce		json
//	@Param			notificationID	path	string	true	"Notification ID (UUID)"
//	@Success		204				"No Content"
//	@Failure		400				{object}	errors.ErrorResponse
//	@Failure		401				{object}	errors.ErrorResponse
//	@Failure		500				{object}	errors.ErrorResponse
//	@ID				markNotificationAsRead
//	@Router			/notifications/{notificationID}/read [post]
//	@Security		BearerAuth
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	notifID, err := uuid.Parse(chi.URLParam(r, "notificationID"))
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid notification ID"))
		return
	}

	if err := h.notifService.MarkAsRead(ctx, notifID, *userID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAllAsRead godoc
//
//	@Summary		Mark all notifications as read
//	@Description	Mark all notifications as read for the authenticated user
//	@Tags			notifications
//	@Produce		json
//	@Success		204	"No Content"
//	@Failure		401	{object}	errors.ErrorResponse
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				markAllNotificationsAsRead
//	@Router			/notifications/read-all [post]
//	@Security		BearerAuth
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	if err := h.notifService.MarkAllAsRead(ctx, *userID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Stream godoc
//
//	@Summary		Stream notifications (SSE)
//	@Description	Open a Server-Sent Events stream for real-time notifications. Pass JWT via Authorization header or ?token= query parameter.
//	@Tags			notifications
//	@Produce		text/event-stream
//	@Param			token	query	string	false	"JWT access token (alternative to Authorization header)"
//	@Success		200		"SSE stream"
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		501		{object}	errors.ErrorResponse	"SSE not available (Redis disabled)"
//	@ID				streamNotifications
//	@Router			/notifications/stream [get]
//	@Security		BearerAuth
func (h *NotificationHandler) Stream(w http.ResponseWriter, r *http.Request) {
	reqCtx := r.Context()

	if h.broker == nil {
		errors.WriteJSON(w, errors.New("SSE_UNAVAILABLE", "real-time notifications not available", http.StatusNotImplemented))
		return
	}

	userID := httputil.GetUserID(reqCtx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	// Check that the response writer supports flushing (required for SSE)
	flusher, ok := w.(http.Flusher)
	if !ok {
		errors.WriteJSON(w, errors.New("SSE_UNSUPPORTED", "streaming not supported", http.StatusInternalServerError))
		return
	}

	// Disable the server-level WriteTimeout for this connection.
	// http.Server.WriteTimeout operates at the TCP layer and cannot be bypassed
	// by context cancellation — it would close the SSE stream after the deadline.
	// time.Time{} (zero) means no deadline.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
		logger.LogError(reqCtx, err, "failed to disable write deadline for SSE stream")
	}

	// Create a long-lived context that is cancelled only when the client disconnects,
	// bypassing the global request timeout middleware.
	// The request context (reqCtx) may have a short deadline from middleware.Timeout;
	// we detach from it and watch the connection close via the ResponseWriter.
	connCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Detect client disconnect via CloseNotify (deprecated but universally supported)
	// or by watching the request context (which Go's HTTP server cancels on disconnect).
	go func() {
		<-reqCtx.Done()
		cancel()
	}()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Subscribe to broker events for this user
	ch, cleanup := h.broker.Subscribe(connCtx, *userID)
	defer cleanup()

	logger.Info(reqCtx, "SSE stream opened", "user_id", userID)
	defer logger.Info(reqCtx, "SSE stream closed", "user_id", userID)

	// Send initial unread count
	if count, err := h.notifService.GetUnreadCount(connCtx, *userID); err == nil {
		data, _ := json.Marshal(map[string]int64{"unread_count": count})
		_, _ = fmt.Fprintf(w, "event: unread_count\ndata: %s\n\n", data)
		flusher.Flush()
	}

	// Stream events until client disconnects or server shuts down
	for {
		select {
		case <-connCtx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, evt.Data)
			flusher.Flush()
		}
	}
}
