package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

// ProfileHandler handles HTTP requests for social profile operations.
type ProfileHandler struct {
	profileService profileService
	followChecker  followChecker
	resolver       userResolver
	storage        storage.Storage
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(profileService *service.UserService, followChecker followChecker, storage storage.Storage) *ProfileHandler {
	return &ProfileHandler{profileService: profileService, followChecker: followChecker, resolver: profileService, storage: storage}
}

// GetMyProfile godoc
//
//	@Summary		Get my profile
//	@Description	Get the authenticated user's social profile
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	dto.UserResponse
//	@Failure		401	{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	errors.ErrorResponse	"User not found"
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				getMyProfile
//	@Router			/me [get]
//	@Security		BearerAuth
func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	u, err := h.profileService.GetMyProfile(ctx, *userID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
	logger.Info(ctx, "get my profile successful", "user_id", userID)
}

// UpdateMyProfile godoc
//
//	@Summary		Update my profile
//	@Description	Update the authenticated user's social profile fields
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			profile	body		dto.UpdateProfileRequest	true	"Profile update data"
//	@Success		200		{object}	dto.UserResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid request body"
//	@Failure		401		{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				updateMyProfile
//	@Router			/me [put]
//	@Security		BearerAuth
func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(ctx, "invalid request body", "error", err)
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	u, err := h.profileService.UpdateMyProfile(ctx, *userID, &req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
	logger.Info(ctx, "profile updated successfully", "user_id", userID)
}

// GetUserProfile godoc
//
//	@Summary		Get user profile
//	@Description	Get a user's public social profile by UUID or username (with ?by=username). Optionally enriches is_following when the viewer is authenticated.
//	@Tags			users
//	@Produce		json
//	@Param			userKey	path		string	true	"User ID (UUID) or username"
//	@Param			by		query		string	false	"Resolve key as 'username' instead of UUID"	Enums(username)
//	@Success		200		{object}	dto.ProfileResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid user identifier"
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				getUserProfile
//	@Router			/users/{userKey}/profile [get]
//	@Security		BearerAuth
func (h *ProfileHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resolved, err := resolveUser(r, "userKey", h.resolver)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	// Reuse entity from username lookup if available, otherwise query by ID.
	u := resolved.User
	if u == nil {
		u, err = h.profileService.GetProfileByUserID(ctx, resolved.ID)
		if err != nil {
			errors.WriteJSON(w, err)
			return
		}
	}

	// Enrich is_following when viewer is authenticated and not viewing own profile
	if viewerID := httputil.GetUserID(ctx); viewerID != nil && *viewerID != resolved.ID {
		ok, err := h.followChecker.IsFollowing(ctx, *viewerID, resolved.ID)
		if err == nil {
			u.IsFollowing = &ok
		}
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToProfileResponse(u, h.storage))
}

// UploadAvatar godoc
//
//	@Summary		Upload avatar
//	@Description	Upload a new avatar image for the authenticated user
//	@Tags			users
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Avatar image (JPEG or PNG, max 10MB)"
//	@Success		200		{object}	dto.UserResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid file"
//	@Failure		401		{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				uploadAvatar
//	@Router			/me/avatar [put]
//	@Security		BearerAuth
func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("file too large or invalid form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("file is required"))
		return
	}
	defer func() { _ = file.Close() }()

	u, err := h.profileService.UploadAvatar(ctx, *userID, file, header.Size, header.Header.Get("Content-Type"), filepath.Ext(header.Filename))
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
	logger.Info(ctx, "avatar uploaded successfully", "user_id", userID)
}

// UploadCover godoc
//
//	@Summary		Upload cover image
//	@Description	Upload a new cover image for the authenticated user
//	@Tags			users
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Cover image (JPEG or PNG, max 10MB)"
//	@Success		200		{object}	dto.UserResponse
//	@Failure		400		{object}	errors.ErrorResponse	"Invalid file"
//	@Failure		401		{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		404		{object}	errors.ErrorResponse	"User not found"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				uploadCover
//	@Router			/me/cover [put]
//	@Security		BearerAuth
func (h *ProfileHandler) UploadCover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("file too large or invalid form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("file is required"))
		return
	}
	defer func() { _ = file.Close() }()

	u, err := h.profileService.UploadCover(ctx, *userID, file, header.Size, header.Header.Get("Content-Type"), filepath.Ext(header.Filename))
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
	logger.Info(ctx, "cover uploaded successfully", "user_id", userID)
}
