package handler

import (
	"net/http"

	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// LikeHandler handles HTTP requests for like operations
type LikeHandler struct {
	likeService likeService
}

// NewLikeHandler creates a new LikeHandler
func NewLikeHandler(likeService likeService) *LikeHandler {
	return &LikeHandler{likeService: likeService}
}

// ToggleLike godoc
//
//	@Summary		Toggle like on a post
//	@Description	Like the post if not yet liked, unlike if already liked. Returns the new like state.
//	@Tags			posts
//	@Produce		json
//	@Param			postID	path		string					true	"Post ID (UUID)"
//	@Success		200		{object}	map[string]bool			"liked: true if now liked, false if now unliked"
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid post ID or self-like"
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				toggleLike
//	@Router			/posts/{postID}/like [post]
//	@Security		BearerAuth
func (h *LikeHandler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	postID, err := parseUUIDParam(r, "postID")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid post ID"))
		return
	}

	liked, err := h.likeService.Toggle(ctx, *userID, postID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "like toggled", "user_id", *userID, "post_id", postID, "liked", liked)
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"liked": liked})
}
