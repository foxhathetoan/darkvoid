package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/post/dto"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// CommentHandler handles HTTP requests for comment operations
type CommentHandler struct {
	commentService commentService
	store          storage.Storage
}

// NewCommentHandler creates a new CommentHandler
func NewCommentHandler(commentService commentService, store storage.Storage) *CommentHandler {
	return &CommentHandler{commentService: commentService, store: store}
}

// CreateComment godoc
//
//	@Summary		Create a comment
//	@Description	Add a comment (or reply) to a post
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		string						true	"Post ID (UUID)"
//	@Param			request	body		dto.CreateCommentRequest	true	"Comment data"
//	@Success		201		{object}	dto.CommentResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				createComment
//	@Router			/posts/{postID}/comments [post]
//	@Security		BearerAuth
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
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

	var req dto.CreateCommentRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		parsed, parseErr := uuid.Parse(*req.ParentID)
		if parseErr != nil {
			errors.WriteJSON(w, errors.NewBadRequestError("Invalid parent comment ID"))
			return
		}
		parentID = &parsed
	}

	mentionIDs, err := parseUUIDs(req.MentionUserIDs)
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid mention_user_ids: "+err.Error()))
		return
	}

	c, err := h.commentService.CreateComment(ctx, postID, *userID, parentID, req.Content, req.MediaKeys, mentionIDs)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "comment created", "comment_id", c.ID, "post_id", postID)
	httputil.WriteJSON(w, http.StatusCreated, dto.ToCommentResponse(c, h.store))
}

// GetComments godoc
//
//	@Summary		Get comments
//	@Description	Get a paginated list of top-level comments for a post
//	@Tags			comments
//	@Produce		json
//	@Param			postID	path		string	true	"Post ID (UUID)"
//	@Param			limit	query		int		false	"Max results (default 20)"
//	@Param			offset	query		int		false	"Offset (default 0)"
//	@Success		200		{object}	dto.CommentListResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getComments
//	@Router			/posts/{postID}/comments [get]
func (h *CommentHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	postID, err := parseUUIDParam(r, "postID")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid post ID"))
		return
	}

	viewerID := httputil.GetUserID(ctx)
	req := pagination.ParseQuery(r)
	comments, pag, err := h.commentService.GetComments(ctx, postID, viewerID, req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToCommentListResponse(comments, pag, h.store))
}

// GetReplies godoc
//
//	@Summary		Get replies
//	@Description	Get a paginated list of replies for a comment
//	@Tags			comments
//	@Produce		json
//	@Param			postID		path		string	true	"Post ID (UUID)"
//	@Param			commentID	path		string	true	"Comment ID (UUID)"
//	@Param			limit		query		int		false	"Max results (default 20)"
//	@Param			offset		query		int		false	"Offset (default 0)"
//	@Success		200			{object}	dto.CommentListResponse
//	@Failure		400			{object}	errors.ErrorResponse
//	@Failure		404			{object}	errors.ErrorResponse
//	@Failure		500			{object}	errors.ErrorResponse
//	@ID				getReplies
//	@Router			/posts/{postID}/comments/{commentID}/replies [get]
func (h *CommentHandler) GetReplies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	commentID, err := parseUUIDParam(r, "commentID")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid comment ID"))
		return
	}

	viewerID := httputil.GetUserID(ctx)
	req := pagination.ParseQuery(r)
	replies, pag, err := h.commentService.GetReplies(ctx, commentID, viewerID, req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToCommentListResponse(replies, pag, h.store))
}

// DeleteComment godoc
//
//	@Summary		Delete a comment
//	@Description	Soft-delete an owned comment
//	@Tags			comments
//	@Produce		json
//	@Param			postID		path	string	true	"Post ID (UUID)"
//	@Param			commentID	path	string	true	"Comment ID (UUID)"
//	@Success		204			"No Content"
//	@Failure		400			{object}	errors.ErrorResponse
//	@Failure		401			{object}	errors.ErrorResponse
//	@Failure		403			{object}	errors.ErrorResponse
//	@Failure		404			{object}	errors.ErrorResponse
//	@Failure		500			{object}	errors.ErrorResponse
//	@ID				deleteComment
//	@Router			/posts/{postID}/comments/{commentID} [delete]
//	@Security		BearerAuth
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
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

	if err := h.commentService.DeleteComment(ctx, commentID, *userID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "comment deleted", "comment_id", commentID)
	w.WriteHeader(http.StatusNoContent)

}
