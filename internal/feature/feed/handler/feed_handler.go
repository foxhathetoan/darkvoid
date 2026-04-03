package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/feed"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// mediaResponse represents one attached media item.
type mediaResponse struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
	URL       string `json:"url"`
	Position  int32  `json:"position"`
}

// authorResponse holds minimal author info embedded in a post response.
type authorResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// postResponse is the feed-local outgoing post representation.
type postResponse struct {
	ID           string          `json:"id"`
	AuthorID     string          `json:"author_id"`
	Author       *authorResponse `json:"author,omitempty"`
	Content      string          `json:"content"`
	Visibility   string          `json:"visibility"`
	Media        []mediaResponse `json:"media"`
	LikeCount    int64           `json:"like_count"`
	CommentCount int64           `json:"comment_count"`
	IsLiked      bool            `json:"is_liked"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

func toPostResponse(p *feedentity.Post, store storage.Storage) postResponse {
	media := make([]mediaResponse, len(p.Media))
	for i, m := range p.Media {
		media[i] = mediaResponse{
			ID:        m.ID.String(),
			MediaType: m.MediaType,
			URL:       store.URL(m.MediaKey),
			Position:  m.Position,
		}
	}

	var author *authorResponse
	if p.Author != nil {
		a := &authorResponse{
			ID:          p.Author.ID.String(),
			Username:    p.Author.Username,
			DisplayName: p.Author.DisplayName,
		}
		if p.Author.AvatarKey != nil {
			url := store.URL(*p.Author.AvatarKey)
			a.AvatarURL = &url
		}
		author = a
	}

	return postResponse{
		ID:           p.ID.String(),
		AuthorID:     p.AuthorID.String(),
		Author:       author,
		Content:      p.Content,
		Visibility:   p.Visibility,
		Media:        media,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		IsLiked:      p.IsLiked,
		CreatedAt:    p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// FeedItemResponse is one item in the scored feed response.
type FeedItemResponse struct {
	postResponse
	Score  float64 `json:"score"`
	Source string  `json:"source"`
}

// FeedResponse is the response for GET /feed.
type FeedResponse struct {
	Data       []FeedItemResponse `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// DiscoverResponse is the response for GET /discover.
type DiscoverResponse struct {
	Data       []postResponse `json:"data"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// feedService defines the methods used by FeedHandler.
// *feedservice.FeedService satisfies this interface.
type feedService interface {
	GetFeed(ctx context.Context, userID uuid.UUID, cursor *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error)
	GetDiscover(ctx context.Context, viewerID *uuid.UUID, cursor *feed.DiscoverCursor, limit int32) ([]*feedentity.Post, *feed.DiscoverCursor, error)
}

// FeedHandler handles HTTP requests for feed operations.
type FeedHandler struct {
	feedService feedService
	store       storage.Storage
}

// NewFeedHandler creates a new FeedHandler.
func NewFeedHandler(feedService feedService, store storage.Storage) *FeedHandler {
	return &FeedHandler{feedService: feedService, store: store}
}

// GetFeed godoc
//
//	@Summary		Get feed
//	@Description	Get a scored, paginated feed of posts mixed from followed users and trending posts
//	@Tags			feed
//	@Produce		json
//	@Param			cursor	query		string	false	"Pagination cursor from previous response"
//	@Success		200		{object}	FeedResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getFeed
//	@Router			/feed [get]
//	@Security		BearerAuth
func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	cursor, err := feed.DecodeFollowingCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid cursor"))
		return
	}

	items, nextCursor, err := h.feedService.GetFeed(ctx, *userID, cursor)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	resp := FeedResponse{
		Data: make([]FeedItemResponse, len(items)),
	}
	for i, item := range items {
		resp.Data[i] = FeedItemResponse{
			postResponse: toPostResponse(item.Post, h.store),
			Score:        item.Score,
			Source:       string(item.Source),
		}
	}
	if nextCursor != nil {
		resp.NextCursor = nextCursor.Encode()
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// GetDiscover godoc
//
//	@Summary		Discover posts
//	@Description	Get a cursor-paginated discovery feed of public posts. No authentication required; if a valid token is provided, is_liked is populated.
//	@Tags			feed
//	@Produce		json
//	@Param			cursor	query		string	false	"Pagination cursor from previous response"
//	@Param			limit	query		int		false	"Max results (default 20)"
//	@Success		200		{object}	DiscoverResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getDiscover
//	@Router			/discover [get]
func (h *FeedHandler) GetDiscover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	viewerID := httputil.GetUserID(ctx)

	cursor, err := feed.DecodeDiscoverCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid cursor"))
		return
	}

	const maxDiscoverLimit = 100
	var limit int32 = 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, parseErr := strconv.Atoi(raw); parseErr == nil && n > 0 {
			limit = min(int32(n), maxDiscoverLimit) //nolint:gosec // n is capped at maxDiscoverLimit (100)
		}
	}

	posts, nextCursor, err := h.feedService.GetDiscover(ctx, viewerID, cursor, limit)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	resp := DiscoverResponse{
		Data: make([]postResponse, len(posts)),
	}
	for i, p := range posts {
		resp.Data[i] = toPostResponse(p, h.store)
	}
	if nextCursor != nil {
		resp.NextCursor = nextCursor.Encode()
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}
