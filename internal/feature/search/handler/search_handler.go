package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/jarviisha/darkvoid/internal/feature/search/dto"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// searchService defines the methods used by SearchHandler.
type searchService interface {
	Search(ctx context.Context, query string, searchType dto.SearchType, limit, offset int32) (*dto.SearchResponse, error)
}

// SearchHandler handles unified search requests.
type SearchHandler struct {
	svc searchService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(svc searchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Search godoc
//
//	@Summary		Unified search
//	@Description	Search across users, posts, and hashtags in a single request.
//	@Description	Use type=all (default) to fetch top results from every category,
//	@Description	or specify type=users|posts|hashtags for a focused search.
//	@Tags			search
//	@Produce		json
//	@Param			q		query		string	true	"Search query (min 2 chars)"
//	@Param			type	query		string	false	"Entity type: all | users | posts | hashtags (default: all)"
//	@Param			limit	query		int		false	"Max results per category (default 20, max 50; ignored for type=all)"
//	@Param			offset	query		int		false	"Offset for pagination (ignored for type=all)"
//	@Success		200		{object}	dto.SearchResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				search
//	@Router			/search [get]
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	query := q.Get("q")

	searchType := dto.SearchType(q.Get("type"))
	if searchType == "" {
		searchType = dto.SearchTypeAll
	}

	var limit int32 = 20
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = int32(n) //nolint:gosec // capped by service layer
		}
	}

	var offset int32
	if raw := q.Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = int32(n) //nolint:gosec // capped by service layer
		}
	}

	resp, err := h.svc.Search(ctx, query, searchType, limit, offset)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// RegisterRoutes mounts search routes onto the given router.
func (h *SearchHandler) RegisterRoutes(r interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}) {
	r.Get("/search", h.Search)
}
