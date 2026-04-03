package handler

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/dto"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// HashtagHandler handles HTTP requests for hashtag operations.
type HashtagHandler struct {
	hashtagService hashtagSvc
	store          storage.Storage
}

// NewHashtagHandler creates a new HashtagHandler.
func NewHashtagHandler(hashtagService hashtagSvc, store storage.Storage) *HashtagHandler {
	return &HashtagHandler{hashtagService: hashtagService, store: store}
}

// GetTrendingHashtags godoc
//
//	@Summary		Trending hashtags
//	@Description	Returns the top trending hashtags based on post usage in the last 24 hours
//	@Tags			hashtags
//	@Produce		json
//	@Success		200	{object}	dto.TrendingHashtagListResponse
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				getTrendingHashtags
//	@Router			/hashtags/trending [get]
func (h *HashtagHandler) GetTrendingHashtags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tags, err := h.hashtagService.GetTrending(ctx)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToTrendingHashtagListResponse(tags))
}

// SearchHashtags godoc
//
//	@Summary		Search hashtags by prefix
//	@Description	Returns up to 10 hashtag names that start with the given prefix (min 2 chars). Results cached 2 minutes.
//	@Tags			hashtags
//	@Produce		json
//	@Param			q	query		string	true	"Prefix to search (min 2 chars, without #)"
//	@Success		200	{object}	dto.HashtagSearchResponse
//	@Failure		400	{object}	errors.ErrorResponse
//	@Failure		429	{object}	errors.ErrorResponse
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				searchHashtags
//	@Router			/hashtags/search [get]
func (h *HashtagHandler) SearchHashtags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if len(prefix) < 2 {
		errors.WriteJSON(w, errors.NewBadRequestError("q must be at least 2 characters"))
		return
	}

	names, err := h.hashtagService.SearchHashtags(ctx, prefix)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.HashtagSearchResponse{Prefix: prefix, Results: names})
}

// GetPostsByHashtag godoc
//
//	@Summary		Posts by hashtag
//	@Description	Returns cursor-paginated public posts for a hashtag
//	@Tags			hashtags
//	@Produce		json
//	@Param			name	path		string	true	"Hashtag name (without #)"
//	@Param			cursor	query		string	false	"Pagination cursor"
//	@Param			limit	query		int		false	"Max results (default 20)"
//	@Success		200		{object}	dto.HashtagPostListResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getPostsByHashtag
//	@Router			/hashtags/{name}/posts [get]
func (h *HashtagHandler) GetPostsByHashtag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := chi.URLParam(r, "name")
	if name == "" {
		errors.WriteJSON(w, errors.NewBadRequestError("hashtag name is required"))
		return
	}

	cursor, err := post.DecodeUserPostCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid cursor"))
		return
	}

	var limit int32 = 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, parseErr := strconv.Atoi(raw); parseErr == nil && n > 0 && n <= math.MaxInt32 {
			limit = int32(n) //nolint:gosec // bounds checked above
		}
	}
	if limit > 100 {
		limit = 100
	}

	viewerID := httputil.GetUserID(ctx)
	posts, nextCursor, err := h.hashtagService.GetPostsByHashtag(ctx, name, viewerID, cursor, limit)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToHashtagPostListResponse(name, posts, nextCursor, h.store))
}
