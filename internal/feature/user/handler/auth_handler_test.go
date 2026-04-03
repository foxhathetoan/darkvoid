package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// --------------------------------------------------------------------------
// Mock: authService
// --------------------------------------------------------------------------

type mockAuthService struct {
	register          func(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	login             func(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	refreshToken      func(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)
	logout            func(ctx context.Context, req *dto.LogoutRequest) error
	logoutAllSessions func(ctx context.Context, userID uuid.UUID) error
	getMe             func(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	changePassword    func(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

func (m *mockAuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	if m.register != nil {
		return m.register(ctx, req)
	}
	return nil, errors.ErrInternal
}
func (m *mockAuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	if m.login != nil {
		return m.login(ctx, req)
	}
	return nil, errors.ErrInternal
}
func (m *mockAuthService) RefreshAccessToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	if m.refreshToken != nil {
		return m.refreshToken(ctx, req)
	}
	return nil, errors.ErrInternal
}
func (m *mockAuthService) Logout(ctx context.Context, req *dto.LogoutRequest) error {
	if m.logout != nil {
		return m.logout(ctx, req)
	}
	return nil
}
func (m *mockAuthService) LogoutAllSessions(ctx context.Context, userID uuid.UUID) error {
	if m.logoutAllSessions != nil {
		return m.logoutAllSessions(ctx, userID)
	}
	return nil
}
func (m *mockAuthService) GetMe(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	if m.getMe != nil {
		return m.getMe(ctx, userID)
	}
	return nil, errors.ErrInternal
}
func (m *mockAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if m.changePassword != nil {
		return m.changePassword(ctx, userID, oldPassword, newPassword)
	}
	return nil
}

// --------------------------------------------------------------------------
// Test helpers
// --------------------------------------------------------------------------

func newAuthHandler(svc authService) *AuthHandler {
	return &AuthHandler{authService: svc, storage: nil}
}

func doPost(t *testing.T, handler http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func doPostWithUser(t *testing.T, handler http.HandlerFunc, path string, body any, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

// --------------------------------------------------------------------------
// Register tests
// --------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockAuthService{
		register: func(_ context.Context, _ *dto.RegisterRequest) (*dto.RegisterResponse, error) {
			return &dto.RegisterResponse{
				UserID:       userID.String(),
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				TokenType:    "Bearer",
			}, nil
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.Register, "/auth/register", dto.RegisterRequest{
		Username:    "johndoe",
		Email:       "john@example.com",
		DisplayName: "John Doe",
		Password:    "SecurePass123",
	})

	assertStatus(t, w, http.StatusCreated)

	var resp dto.RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UserID != userID.String() {
		t.Errorf("expected user_id %q, got %q", userID.String(), resp.UserID)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token_type Bearer, got %q", resp.TokenType)
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/register", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, req)

	assertStatus(t, w, http.StatusBadRequest)
	assertErrorCode(t, w, "BAD_REQUEST")
}

func TestRegister_ServiceConflict(t *testing.T) {
	svc := &mockAuthService{
		register: func(_ context.Context, _ *dto.RegisterRequest) (*dto.RegisterResponse, error) {
			return nil, errors.NewConflictError("username already exists")
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.Register, "/auth/register", dto.RegisterRequest{
		Username: "existing", Email: "e@e.com", Password: "Pass123",
	})

	assertStatus(t, w, http.StatusConflict)
	assertErrorCode(t, w, "CONFLICT")
}

// --------------------------------------------------------------------------
// Login tests
// --------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	svc := &mockAuthService{
		login: func(_ context.Context, _ *dto.LoginRequest) (*dto.LoginResponse, error) {
			return &dto.LoginResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				TokenType:    "Bearer",
			}, nil
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.Login, "/auth/login", dto.LoginRequest{
		Username: "johndoe",
		Password: "SecurePass123",
	})

	assertStatus(t, w, http.StatusOK)

	var resp dto.LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken != "access-token" {
		t.Errorf("expected access_token, got %q", resp.AccessToken)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	assertStatus(t, w, http.StatusBadRequest)
	assertErrorCode(t, w, "BAD_REQUEST")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := &mockAuthService{
		login: func(_ context.Context, _ *dto.LoginRequest) (*dto.LoginResponse, error) {
			return nil, errors.NewUnauthorizedError("invalid username or password")
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.Login, "/auth/login", dto.LoginRequest{
		Username: "johndoe", Password: "wrong",
	})

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestLogin_AccountDeactivated(t *testing.T) {
	svc := &mockAuthService{
		login: func(_ context.Context, _ *dto.LoginRequest) (*dto.LoginResponse, error) {
			return nil, errors.NewForbiddenError("user account is deactivated")
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.Login, "/auth/login", dto.LoginRequest{
		Username: "johndoe", Password: "SecurePass123",
	})

	assertStatus(t, w, http.StatusForbidden)
	assertErrorCode(t, w, "FORBIDDEN")
}

// --------------------------------------------------------------------------
// RefreshToken tests
// --------------------------------------------------------------------------

func TestRefreshToken_Success(t *testing.T) {
	svc := &mockAuthService{
		refreshToken: func(_ context.Context, _ *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
			return &dto.RefreshTokenResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				TokenType:    "Bearer",
			}, nil
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.RefreshToken, "/auth/refresh", dto.RefreshTokenRequest{
		RefreshToken: "old-refresh-token",
	})

	assertStatus(t, w, http.StatusOK)

	var resp dto.RefreshTokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken != "new-access-token" {
		t.Errorf("expected new access token, got %q", resp.AccessToken)
	}
}

func TestRefreshToken_InvalidJSON(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", bytes.NewBufferString("bad json"))
	w := httptest.NewRecorder()
	h.RefreshToken(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc := &mockAuthService{
		refreshToken: func(_ context.Context, _ *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
			return nil, errors.NewUnauthorizedError("invalid refresh token")
		},
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.RefreshToken, "/auth/refresh", dto.RefreshTokenRequest{
		RefreshToken: "bad-token",
	})

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

// --------------------------------------------------------------------------
// Logout tests
// --------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	svc := &mockAuthService{
		logout: func(_ context.Context, _ *dto.LogoutRequest) error { return nil },
	}
	h := newAuthHandler(svc)

	w := doPost(t, h.Logout, "/auth/logout", dto.LogoutRequest{
		RefreshToken: "my-token",
	})

	assertStatus(t, w, http.StatusOK)

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["message"] == "" {
		t.Error("expected message in response")
	}
}

func TestLogout_InvalidJSON(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout", bytes.NewBufferString("bad json"))
	w := httptest.NewRecorder()
	h.Logout(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// --------------------------------------------------------------------------
// GetMe tests
// --------------------------------------------------------------------------

func TestGetMe_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockAuthService{
		getMe: func(_ context.Context, id uuid.UUID) (*entity.User, error) {
			return &entity.User{
				ID:          id,
				Username:    "johndoe",
				Email:       "john@example.com",
				IsActive:    true,
				DisplayName: "John Doe",
				CreatedAt:   time.Now(),
			}, nil
		},
	}
	h := newAuthHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.GetMe(w, req)

	assertStatus(t, w, http.StatusOK)

	var resp dto.UserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Username != "johndoe" {
		t.Errorf("expected username johndoe, got %q", resp.Username)
	}
}

func TestGetMe_NotAuthenticated(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()
	h.GetMe(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestGetMe_UserNotFound(t *testing.T) {
	userID := uuid.New()
	svc := &mockAuthService{
		getMe: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}
	h := newAuthHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.GetMe(w, req)

	assertStatus(t, w, http.StatusNotFound)
	assertErrorCode(t, w, "NOT_FOUND")
}

// --------------------------------------------------------------------------
// LogoutAllSessions tests
// --------------------------------------------------------------------------

func TestLogoutAllSessions_Success(t *testing.T) {
	userID := uuid.New()
	var passedUserID uuid.UUID
	svc := &mockAuthService{
		logoutAllSessions: func(_ context.Context, id uuid.UUID) error {
			passedUserID = id
			return nil
		},
	}
	h := newAuthHandler(svc)

	w := doPostWithUser(t, h.LogoutAllSessions, "/auth/logout-all", nil, userID)

	assertStatus(t, w, http.StatusOK)
	if passedUserID != userID {
		t.Errorf("expected userID %v passed to service, got %v", userID, passedUserID)
	}
}

func TestLogoutAllSessions_NotAuthenticated(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout-all", nil)
	w := httptest.NewRecorder()
	h.LogoutAllSessions(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

// --------------------------------------------------------------------------
// ChangePassword tests
// --------------------------------------------------------------------------

func TestChangePassword_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockAuthService{
		changePassword: func(_ context.Context, _ uuid.UUID, _, _ string) error { return nil },
	}
	h := newAuthHandler(svc)

	w := doPostWithUser(t, h.ChangePassword, "/auth/password", dto.ChangePasswordRequest{
		OldPassword: "OldPass123",
		NewPassword: "NewPass456",
	}, userID)

	assertStatus(t, w, http.StatusOK)

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["message"] == "" {
		t.Error("expected message in response")
	}
}

func TestChangePassword_NotAuthenticated(t *testing.T) {
	h := newAuthHandler(&mockAuthService{})

	b, _ := json.Marshal(dto.ChangePasswordRequest{OldPassword: "old", NewPassword: "new"})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/auth/password", bytes.NewReader(b))
	w := httptest.NewRecorder()
	h.ChangePassword(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestChangePassword_InvalidJSON(t *testing.T) {
	userID := uuid.New()
	h := newAuthHandler(&mockAuthService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/auth/password", bytes.NewBufferString("bad json"))
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.ChangePassword(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	userID := uuid.New()
	svc := &mockAuthService{
		changePassword: func(_ context.Context, _ uuid.UUID, _, _ string) error {
			return errors.NewUnauthorizedError("invalid old password")
		},
	}
	h := newAuthHandler(svc)

	w := doPostWithUser(t, h.ChangePassword, "/auth/password", dto.ChangePasswordRequest{
		OldPassword: "WrongOld123",
		NewPassword: "NewPass456",
	}, userID)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}
