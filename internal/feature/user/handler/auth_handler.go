package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

const refreshTokenCookieName = "refresh_token"

// isMobileClient returns true when the request originates from a mobile client.
// Mobile clients send X-Client-Type: mobile and manage the refresh token themselves.
// Web clients rely on HttpOnly cookies set by the server.
func isMobileClient(r *http.Request) bool {
	return r.Header.Get("X-Client-Type") == "mobile"
}

// AuthHandler handles HTTP requests for authentication operations.
type AuthHandler struct {
	authService  authService
	storage      storage.Storage
	secureCookie bool
}

// NewAuthHandler creates a new auth handler.
// secureCookie should be true in production (HTTPS) and false in development (HTTP).
func NewAuthHandler(authService *service.AuthService, storage storage.Storage, secureCookie bool) *AuthHandler {
	return &AuthHandler{authService: authService, storage: storage, secureCookie: secureCookie}
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string, expiry time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(expiry.Seconds()),
	})
}

func (h *AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// Register godoc
//
//	@Summary		Register a new account
//	@Description	Create a new user account and return tokens (user is immediately logged in).
//	@Description	Web clients (default): refresh_token is set as an HttpOnly cookie; omitted from response body.
//	@Description	Mobile clients (X-Client-Type: mobile): refresh_token is returned in the response body.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			X-Client-Type	header		string				false	"Set to 'mobile' to receive refresh_token in response body instead of HttpOnly cookie"
//	@Param			request			body		dto.RegisterRequest	true	"Registration data"
//	@Success		201				{object}	dto.RegisterResponse
//	@Failure		400				{object}	errors.ErrorResponse	"Invalid request body"
//	@Failure		409				{object}	errors.ErrorResponse	"Username or email already exists"
//	@Failure		500				{object}	errors.ErrorResponse
//	@ID				register
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(ctx, "invalid request body", "error", err)
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	resp, err := h.authService.Register(ctx, &req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if !isMobileClient(r) {
		h.setRefreshTokenCookie(w, resp.RefreshToken, time.Duration(resp.RefreshExpiresIn)*time.Second)
		resp.RefreshToken = ""
	}

	httputil.WriteJSON(w, http.StatusCreated, resp)
	logger.Info(ctx, "user registered successfully", "user_id", resp.UserID)
}

// Login godoc
//
//	@Summary		User login
//	@Description	Authenticate user with username and password, returns access and refresh tokens.
//	@Description	Web clients (default): refresh_token is set as an HttpOnly cookie; omitted from response body.
//	@Description	Mobile clients (X-Client-Type: mobile): refresh_token is returned in the response body.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			X-Client-Type	header		string				false	"Set to 'mobile' to receive refresh_token in response body instead of HttpOnly cookie"
//	@Param			credentials		body		dto.LoginRequest	true	"Login credentials"
//	@Success		200				{object}	dto.LoginResponse
//	@Failure		400				{object}	errors.ErrorResponse	"Invalid request body"
//	@Failure		401				{object}	errors.ErrorResponse	"Invalid credentials"
//	@Failure		403				{object}	errors.ErrorResponse	"Account deactivated"
//	@Failure		500				{object}	errors.ErrorResponse
//	@ID				login
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	resp, err := h.authService.Login(ctx, &req)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if !isMobileClient(r) {
		h.setRefreshTokenCookie(w, resp.RefreshToken, time.Duration(resp.RefreshExpiresIn)*time.Second)
		resp.RefreshToken = ""
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// RefreshToken godoc
//
//	@Summary		Refresh access token
//	@Description	Exchange a valid refresh token for a new access token and refresh token.
//	@Description	Web clients (default): refresh_token is read from the HttpOnly cookie automatically; request body is not required.
//	@Description	Mobile clients (X-Client-Type: mobile): refresh_token must be provided in the request body.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			X-Client-Type	header		string					false	"Set to 'mobile' to send/receive refresh_token in request/response body"
//	@Param			token			body		dto.RefreshTokenRequest	false	"Refresh token (mobile clients only)"
//	@Success		200				{object}	dto.RefreshTokenResponse
//	@Failure		400				{object}	errors.ErrorResponse	"Invalid request body"
//	@Failure		401				{object}	errors.ErrorResponse	"Invalid or expired refresh token"
//	@Failure		500				{object}	errors.ErrorResponse
//	@ID				refreshToken
//	@Router			/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	mobile := isMobileClient(r)

	var tokenString string
	if mobile {
		var req dto.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
			return
		}
		tokenString = req.RefreshToken
	} else {
		cookie, err := r.Cookie(refreshTokenCookieName)
		if err != nil {
			errors.WriteJSON(w, errors.NewUnauthorizedError("missing refresh token"))
			return
		}
		tokenString = cookie.Value
	}

	resp, err := h.authService.RefreshAccessToken(ctx, &dto.RefreshTokenRequest{RefreshToken: tokenString})
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if !mobile {
		h.setRefreshTokenCookie(w, resp.RefreshToken, time.Duration(resp.RefreshExpiresIn)*time.Second)
		resp.RefreshToken = ""
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// Logout godoc
//
//	@Summary		User logout
//	@Description	Revoke a refresh token to logout user from current session.
//	@Description	Web clients (default): refresh_token is read from the HttpOnly cookie automatically; request body is not required. Cookie is cleared on success.
//	@Description	Mobile clients (X-Client-Type: mobile): refresh_token must be provided in the request body.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			X-Client-Type	header		string						false	"Set to 'mobile' to send refresh_token in request body"
//	@Param			token			body		dto.LogoutRequest			false	"Refresh token to revoke (mobile clients only)"
//	@Success		200				{object}	httputil.MessageResponse	"Logged out successfully"
//	@Failure		400				{object}	errors.ErrorResponse		"Invalid request body"
//	@Failure		401				{object}	errors.ErrorResponse		"Missing refresh token cookie (web clients)"
//	@Failure		500				{object}	errors.ErrorResponse
//	@ID				logout
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	mobile := isMobileClient(r)

	var tokenString string
	if mobile {
		var req dto.LogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
			return
		}
		tokenString = req.RefreshToken
	} else {
		cookie, err := r.Cookie(refreshTokenCookieName)
		if err != nil {
			errors.WriteJSON(w, errors.NewUnauthorizedError("missing refresh token"))
			return
		}
		tokenString = cookie.Value
	}

	if err := h.authService.Logout(ctx, &dto.LogoutRequest{RefreshToken: tokenString}); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if !mobile {
		h.clearRefreshTokenCookie(w)
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("Logged out successfully"))
}

// GetMe godoc
//
//	@Summary		Get current user
//	@Description	Get the authenticated user's information
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	dto.UserResponse
//	@Failure		401	{object}	errors.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	errors.ErrorResponse	"User not found"
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				getMe
//	@Router			/auth/me [get]
//	@Security		BearerAuth
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	u, err := h.authService.GetMe(ctx, *userID)
	if err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToUserResponse(u, h.storage))
}

// LogoutAllSessions godoc
//
//	@Summary		Logout all sessions
//	@Description	Revoke all refresh tokens for the authenticated user
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	httputil.MessageResponse	"All sessions logged out successfully"
//	@Failure		401	{object}	errors.ErrorResponse		"Unauthorized"
//	@Failure		500	{object}	errors.ErrorResponse
//	@ID				logoutAllSessions
//	@Router			/auth/logout-all [post]
//	@Security		BearerAuth
func (h *AuthHandler) LogoutAllSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	if err := h.authService.LogoutAllSessions(ctx, *userID); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	if !isMobileClient(r) {
		h.clearRefreshTokenCookie(w)
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("All sessions logged out successfully"))
}

// ChangePassword godoc
//
//	@Summary		Change password
//	@Description	Change the authenticated user's password (requires old password)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ChangePasswordRequest	true	"Password change data"
//	@Success		200		{object}	httputil.MessageResponse	"Password changed successfully"
//	@Failure		400		{object}	errors.ErrorResponse		"Invalid request"
//	@Failure		401		{object}	errors.ErrorResponse		"Invalid old password or not authenticated"
//	@Failure		500		{object}	errors.ErrorResponse
//	@ID				changePassword
//	@Router			/auth/password [put]
//	@Security		BearerAuth
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := httputil.GetUserID(ctx)
	if userID == nil {
		errors.WriteJSON(w, errors.NewUnauthorizedError("user not authenticated"))
		return
	}

	var req dto.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteJSON(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if err := h.authService.ChangePassword(ctx, *userID, req.OldPassword, req.NewPassword); err != nil {
		errors.WriteJSON(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewMessageResponse("Password changed successfully. All sessions have been logged out."))
}
