package handler

import (
	"net/http"

	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// CommentLikeHandler handles HTTP requests for comment like operations.
type CommentLikeHandler struct {
	commentLikeService commentLikeService
}

// NewCommentLikeHandler creates a new CommentLikeHandler.
func NewCommentLikeHandler(svc commentLikeService) *CommentLikeHandler {
	return &CommentLikeHandler{commentLikeService: svc}
}

// ToggleCommentLike godoc
//
//	@Summary		Toggle like on a comment
//	@Description	Like the comment if not yet liked, unlike if already liked. Returns the new like state.
//	@Tags			comments
//	@Produce		json
//	@Param			postID		path		string			true	"Post ID (UUID)"
//	@Param			commentID	path		string			true	"Comment ID (UUID)"
//	@Success		200			{object}	map[string]bool	"liked: true if now liked, false if now unliked"
//	@Failure		400			{object}	errors.ErrorResponse
//	@Failure		401			{object}	errors.ErrorResponse
//	@Failure		404			{object}	errors.ErrorResponse
//	@Failure		500			{object}	errors.ErrorResponse
//	@ID				toggleCommentLike
//	@Router			/posts/{postID}/comments/{commentID}/like [post]
//	@Security		BearerAuth
func (h *CommentLikeHandler) ToggleCommentLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	commentID, err := parseUUIDParam(r, "commentID")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid comment ID"))
		return
	}

	liked, err := h.commentLikeService.Toggle(ctx, *userID, commentID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "comment like toggled", "user_id", *userID, "comment_id", commentID, "liked", liked)
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"liked": liked})
}
