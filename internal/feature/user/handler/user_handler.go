package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// UserHandler handles HTTP requests for user management operations.
type UserHandler struct {
	userService userService
	resolver    userResolver
	storage     storage.Storage
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userService *service.UserService, resolver userResolver, storage storage.Storage) *UserHandler {
	return &UserHandler{userService: userService, resolver: resolver, storage: storage}
}

// GetUser godoc
//
//	@Summary		Get user by ID or username
//	@Description	Retrieve a user's information by UUID or username (with ?by=username)
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path		string	true	"User ID (UUID) or username"
//	@Param			by		query		string	false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Success		200		{object}	dto.UserResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid user identifier"
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getUser
//	@Router			/users/{userKey} [get]
//	@Security		BearerAuth
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	u := resolved.User
	if u == nil {
		u, err = h.userService.GetUserByID(ctx, resolved.ID)
		if err != nil {
			errors.WriteJSON(w, err)
			return
		}
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
}

// UpdateUser godoc
//
//	@Summary		Update user account
//	@Description	Update user account fields (e.g. email) by UUID or username (with ?by=username)
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userKey	path		string					true	"User ID (UUID) or username"
//	@Param			by		query		string					false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Param			user	body		dto.UpdateUserRequest	true	"User update data"
//	@Success		200		{object}	dto.UserResponse
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		409		{object}	errors.ErrorResponse	"Email already exists"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				updateUser
//	@Router			/users/{userKey} [put]
//	@Security		BearerAuth
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	currentUserID := httputil.GetUserID(ctx)
	if currentUserID == nil || resolved.ID != *currentUserID {
		errors.WriteJSON(w, errors.NewForbiddenError("you can only update your own account"))
		return
	}

	var req dto.UpdateUserRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	u, err := h.userService.UpdateUser(ctx, resolved.ID, &req, currentUserID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
	logger.Info(ctx, "user updated successfully", "user_id", resolved.ID)
}

// DeactivateUser godoc
//
//	@Summary		Deactivate user
//	@Description	Soft delete a user account (sets is_active to false) by UUID or username (with ?by=username)
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path		string						true	"User ID (UUID) or username"
//	@Param			by		query		string						false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Success		200		{object}	httputil.MessageResponse	"User deactivated successfully"
//	@Failure		400		{object}	errors.ErrorResponse
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				deactivateUser
//	@Router			/users/{userKey} [delete]
//	@Security		BearerAuth
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	currentUserID := httputil.GetUserID(ctx)
	if currentUserID == nil || resolved.ID != *currentUserID {
		errors.WriteJSON(w, errors.NewForbiddenError("you can only deactivate your own account"))
		return
	}

	if err := h.userService.DeactivateUser(ctx, resolved.ID, currentUserID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("User deactivated successfully"))
	logger.Info(ctx, "user deactivated successfully", "user_id", resolved.ID)
}

// resolvedUser holds the result of resolveUser.
// When resolved by username, User is populated (avoiding a second DB query).
type resolvedUser struct {
	ID   uuid.UUID
	User *entity.User // non-nil only when resolved by username
}

// resolveUser resolves the path parameter to a user ID (and optionally the full entity).
// If ?by=username, the param is treated as a username and looked up via resolver.
// Otherwise it is parsed as a UUID directly.
func resolveUser(r *http.Request, _ string, resolver userResolver) (resolvedUser, error) {
	raw := chi.URLParam(r, "userKey")
	if r.URL.Query().Get("by") == "username" {
		u, err := resolver.GetUserByUsername(r.Context(), raw)
		if err != nil {
			return resolvedUser{}, err
		}
		return resolvedUser{ID: u.ID, User: u}, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return resolvedUser{}, errors.NewBadRequestError("invalid user identifier")
	}
	return resolvedUser{ID: id}, nil
}
