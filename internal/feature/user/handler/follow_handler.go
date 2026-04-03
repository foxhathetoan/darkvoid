package handler

import (
	"net/http"

	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// FollowHandler handles HTTP requests for follow/unfollow operations.
type FollowHandler struct {
	followService followService
	resolver      userResolver
}

// NewFollowHandler creates a new FollowHandler.
func NewFollowHandler(followService *service.FollowService, resolver userResolver) *FollowHandler {
	return &FollowHandler{followService: followService, resolver: resolver}
}

// Follow godoc
//
//	@Summary		Follow a user
//	@Description	Follow a user by UUID or username (with ?by=username). No-op if already following.
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path	string	true	"User ID (UUID) or username"
//	@Param			by		query	string	false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Success		204		"No Content"
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid user identifier or self-follow"
//	@Failure		401		{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				followUser
//	@Router			/users/{userKey}/follow [post]
//	@Security		BearerAuth
func (h *FollowHandler) Follow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	followerID := httputil.GetUserID(ctx)
	if followerID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if err := h.followService.Follow(ctx, *followerID, resolved.ID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "follow successful", "follower", followerID, "followee", resolved.ID)
	w.WriteHeader(http.StatusNoContent)
}

// Unfollow godoc
//
//	@Summary		Unfollow a user
//	@Description	Unfollow a user by UUID or username (with ?by=username). No-op if not following.
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path	string	true	"User ID (UUID) or username"
//	@Param			by		query	string	false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Success		204		"No Content"
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid user identifier or self-unfollow"
//	@Failure		401		{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				unfollowUser
//	@Router			/users/{userKey}/follow [delete]
//	@Security		BearerAuth
func (h *FollowHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	followerID := httputil.GetUserID(ctx)
	if followerID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if err := h.followService.Unfollow(ctx, *followerID, resolved.ID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	logger.Info(ctx, "unfollow successful", "follower", followerID, "followee", resolved.ID)
	w.WriteHeader(http.StatusNoContent)
}

// GetFollowers godoc
//
//	@Summary		Get followers
//	@Description	Get a paginated list of users who follow the specified user by UUID or username (with ?by=username)
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path		string	true	"User ID (UUID) or username"
//	@Param			by		query		string	false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Param			limit	query		int		false	"Max results (default 20)"
//	@Param			offset	query		int		false	"Offset (default 0)"
//	@Success		200		{object}	dto.FollowListResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid user identifier"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getFollowers
//	@Router			/users/{userKey}/followers [get]
func (h *FollowHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	req := pagination.ParseQuery(r)
	follows, pag, err := h.followService.GetFollowers(ctx, resolved.ID, req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	data := make([]dto.FollowResponse, len(follows))
	for i, f := range follows {
		data[i] = dto.ToFollowerResponse(f)
	}

	httputil.WriteJSON(w, http.StatusOK, dto.FollowListResponse{Data: data, Pagination: pag})
}

// GetFollowing godoc
//
//	@Summary		Get following
//	@Description	Get a paginated list of users that the specified user follows by UUID or username (with ?by=username)
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path		string	true	"User ID (UUID) or username"
//	@Param			by		query		string	false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Param			limit	query		int		false	"Max results (default 20)"
//	@Param			offset	query		int		false	"Offset (default 0)"
//	@Success		200		{object}	dto.FollowListResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid user identifier"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getFollowing
//	@Router			/users/{userKey}/following [get]
func (h *FollowHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	req := pagination.ParseQuery(r)
	follows, pag, err := h.followService.GetFollowing(ctx, resolved.ID, req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	data := make([]dto.FollowResponse, len(follows))
	for i, f := range follows {
		data[i] = dto.ToFollowingResponse(f)
	}

	httputil.WriteJSON(w, http.StatusOK, dto.FollowListResponse{Data: data, Pagination: pag})
}
