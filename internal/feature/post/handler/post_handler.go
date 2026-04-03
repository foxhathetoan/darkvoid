package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/dto"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// PostHandler handles HTTP requests for post operations
type PostHandler struct {
	postService postService
	store       storage.Storage
}

// NewPostHandler creates a new PostHandler
func NewPostHandler(postService postService, store storage.Storage) *PostHandler {
	return &PostHandler{postService: postService, store: store}
}

// CreatePost godoc
//
//	@Summary		Create a post
//	@Description	Create a new post with optional media attachments
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreatePostRequest	true	"Post data"
//	@Success		201		{object}	dto.PostResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				createPost
//	@Router			/posts [post]
//	@Security		BearerAuth
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	var req dto.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	visibility := entity.Visibility(req.Visibility)
	if visibility == "" {
		visibility = entity.VisibilityPublic
	}

	mentionIDs, err := parseUUIDs(req.MentionUserIDs)
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid mention_user_ids: "+err.Error()))
		return
	}

	p, err := h.postService.CreatePost(ctx, *userID, req.Content, visibility, req.MediaKeys, mentionIDs, req.Tags)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "post created", "post_id", p.ID)
	httputil.WriteJSON(w, http.StatusCreated, dto.ToPostResponse(p, h.store))
}

// GetPost godoc
//
//	@Summary		Get a post
//	@Description	Get a post by ID
//	@Tags			posts
//	@Produce		json
//	@Param			postID	path		string	true	"Post ID (UUID)"
//	@Success		200		{object}	dto.PostResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getPost
//	@Router			/posts/{postID} [get]
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	postID, err := parseUUIDParam(r, "postID")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid post ID"))
		return
	}

	viewerID := httputil.GetUserID(ctx)
	p, err := h.postService.GetPost(ctx, postID, viewerID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToPostResponse(p, h.store))
}

// UpdatePost godoc
//
//	@Summary		Update a post
//	@Description	Update content or visibility of an owned post
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		string					true	"Post ID (UUID)"
//	@Param			request	body		dto.UpdatePostRequest	true	"Updated fields"
//	@Success		200		{object}	dto.PostResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		403		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				updatePost
//	@Router			/posts/{postID} [put]
//	@Security		BearerAuth
func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
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

	var req dto.UpdatePostRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	mentionIDs, err := parseUUIDs(req.MentionUserIDs)
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("invalid mention_user_ids: "+err.Error()))
		return
	}

	p, err := h.postService.UpdatePost(ctx, postID, *userID, req.Content, entity.Visibility(req.Visibility), mentionIDs, req.Tags)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToPostResponse(p, h.store))
}

// DeletePost godoc
//
//	@Summary		Delete a post
//	@Description	Soft-delete an owned post
//	@Tags			posts
//	@Produce		json
//	@Param			postID	path	string	true	"Post ID (UUID)"
//	@Success		204		"No Content"
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		401		{object}	errors.ErrorResponse
//	@Failure		403		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				deletePost
//	@Router			/posts/{postID} [delete]
//	@Security		BearerAuth
func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
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

	if err := h.postService.DeletePost(ctx, postID, *userID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserPosts godoc
//
//	@Summary		Get user posts
//	@Description	Get a cursor-paginated list of posts by a specific user
//	@Tags			posts
//	@Produce		json
//	@Param			userID		path		string	true	"Author user ID (UUID)"
//	@Param			cursor		query		string	false	"Pagination cursor from previous response"
//	@Param			limit		query		int		false	"Max results (default 20)"
//	@Param			visibility	query		string	false	"Filter by visibility: public, followers, private"
//	@Success		200			{object}	dto.PostListResponse
//	@Failure		400			{object}	errors.ErrorResponse
//	@Failure		500			{object}	errors.ErrorResponse
//	@ID				getUserPosts
//	@Router			/users/{userID}/posts [get]
func (h *PostHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authorID, err := parseUUIDParam(r, "userID")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid user ID"))
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

	visibility := r.URL.Query().Get("visibility")

	viewerID := httputil.GetUserID(ctx)
	posts, nextCursor, err := h.postService.GetUserPosts(ctx, authorID, viewerID, cursor, visibility, limit)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToPostListResponse(posts, nextCursor, h.store))
}

// parseUUIDParam parses a UUID from a chi URL parameter.
func parseUUIDParam(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}

// parseUUIDs converts a slice of UUID strings to []uuid.UUID, returning an error for any invalid value.
func parseUUIDs(raw []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID %q", s)
		}
		out = append(out, id)
	}
	return out, nil
}
