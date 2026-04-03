package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/jwt"
)

// --------------------------------------------------------------------------
// Mock: refreshTokenRepo
// --------------------------------------------------------------------------

type mockRefreshTokenRepo struct {
	create              func(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) (*entity.RefreshToken, error)
	getByToken          func(ctx context.Context, token string) (*entity.RefreshToken, error)
	revoke              func(ctx context.Context, token string) error
	revokeAllUserTokens func(ctx context.Context, userID uuid.UUID) error
	deleteExpired       func(ctx context.Context) error
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) (*entity.RefreshToken, error) {
	if m.create != nil {
		return m.create(ctx, token, userID, expiresAt)
	}
	return &entity.RefreshToken{
		ID:        uuid.New(),
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}
func (m *mockRefreshTokenRepo) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	if m.getByToken != nil {
		return m.getByToken(ctx, token)
	}
	return nil, errors.New("NOT_FOUND", "token not found", 404)
}
func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, token string) error {
	if m.revoke != nil {
		return m.revoke(ctx, token)
	}
	return nil
}
func (m *mockRefreshTokenRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	if m.revokeAllUserTokens != nil {
		return m.revokeAllUserTokens(ctx, userID)
	}
	return nil
}
func (m *mockRefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	if m.deleteExpired != nil {
		return m.deleteExpired(ctx)
	}
	return nil
}

// --------------------------------------------------------------------------
// Test helpers
// --------------------------------------------------------------------------

func newTestJWT(t *testing.T) *jwt.Service {
	t.Helper()
	svc, err := jwt.NewService(jwt.Config{
		Secret: []byte("test-secret-key-32-bytes-minimum!!"),
		Issuer: "test",
		Expiry: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}
	return svc
}

func newValidToken(t *testing.T, userID uuid.UUID) (*entity.RefreshToken, *mockRefreshTokenRepo) { //nolint:unparam // first return used by callers for setup context
	t.Helper()
	tok := &entity.RefreshToken{
		ID:        uuid.New(),
		Token:     "valid-refresh-token",
		UserID:    userID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		IsRevoked: false,
	}
	repo := &mockRefreshTokenRepo{
		getByToken: func(_ context.Context, _ string) (*entity.RefreshToken, error) {
			return tok, nil
		},
	}
	return tok, repo
}

func newAuthService(userRepo userRepo, rtRepo refreshTokenRepo, jwtSvc *jwt.Service) *AuthService {
	rtSvc := &RefreshTokenService{repo: rtRepo, expiry: 7 * 24 * time.Hour}
	userSvc := &UserService{userRepo: userRepo, storage: nil}
	return &AuthService{
		userRepo:            userRepo,
		userService:         userSvc,
		jwtService:          jwtSvc,
		refreshTokenService: rtSvc,
	}
}

// --------------------------------------------------------------------------
// Login tests
// --------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	userID := uuid.New()
	hashedPw, _ := hashPassword("SecurePass123")
	user := &entity.User{
		ID:           userID,
		Username:     "johndoe",
		PasswordHash: hashedPw,
		IsActive:     true,
		DisplayName:  "John Doe",
	}

	userRepo := &mockUserRepo{
		getUserByUsername: func(_ context.Context, _ string) (*entity.User, error) {
			return user, nil
		},
	}
	rtRepo := &mockRefreshTokenRepo{}
	jwtSvc := newTestJWT(t)

	svc := newAuthService(userRepo, rtRepo, jwtSvc)

	resp, err := svc.Login(context.Background(), &dto.LoginRequest{
		Username: "johndoe",
		Password: "SecurePass123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %q", resp.TokenType)
	}
}

func TestLogin_EmptyUsername(t *testing.T) {
	svc := newAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Username: "",
		Password: "SecurePass123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "BAD_REQUEST")
}

func TestLogin_EmptyPassword(t *testing.T) {
	svc := newAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Username: "johndoe",
		Password: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "BAD_REQUEST")
}

func TestLogin_WrongPassword(t *testing.T) {
	hashedPw, _ := hashPassword("CorrectPass123")
	userRepo := &mockUserRepo{
		getUserByUsername: func(_ context.Context, _ string) (*entity.User, error) {
			return &entity.User{
				ID:           uuid.New(),
				Username:     "johndoe",
				PasswordHash: hashedPw,
				IsActive:     true,
				DisplayName:  "John Doe",
			}, nil
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Username: "johndoe",
		Password: "WrongPass123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "UNAUTHORIZED")
}

func TestLogin_UserNotFound_ReturnsUnauthorized(t *testing.T) {
	// Should not reveal whether user exists
	userRepo := &mockUserRepo{
		getUserByUsername: func(_ context.Context, _ string) (*entity.User, error) {
			return nil, errors.New("NOT_FOUND", "not found", 404)
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Username: "nobody",
		Password: "SomePass123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Must return UNAUTHORIZED (not NOT_FOUND) to avoid user enumeration
	assertServiceErrorCode(t, err, "UNAUTHORIZED")
}

func TestLogin_InactiveUser(t *testing.T) {
	hashedPw, _ := hashPassword("SecurePass123")
	userRepo := &mockUserRepo{
		getUserByUsername: func(_ context.Context, _ string) (*entity.User, error) {
			return &entity.User{
				ID:           uuid.New(),
				Username:     "johndoe",
				PasswordHash: hashedPw,
				IsActive:     false,
				DisplayName:  "John Doe",
			}, nil
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Username: "johndoe",
		Password: "SecurePass123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "FORBIDDEN")
}

// --------------------------------------------------------------------------
// RefreshAccessToken tests
// --------------------------------------------------------------------------

func TestRefreshAccessToken_Success(t *testing.T) {
	userID := uuid.New()
	_, rtRepo := newValidToken(t, userID)
	rtRepo.revoke = func(_ context.Context, _ string) error { return nil }

	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(userID), nil
		},
	}
	svc := newAuthService(userRepo, rtRepo, newTestJWT(t))

	resp, err := svc.RefreshAccessToken(context.Background(), &dto.RefreshTokenRequest{
		RefreshToken: "valid-refresh-token",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
}

func TestRefreshAccessToken_EmptyToken(t *testing.T) {
	svc := newAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.RefreshAccessToken(context.Background(), &dto.RefreshTokenRequest{
		RefreshToken: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "BAD_REQUEST")
}

func TestRefreshAccessToken_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	expiredToken := &entity.RefreshToken{
		ID:        uuid.New(),
		Token:     "expired-token",
		UserID:    userID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // already expired
		IsRevoked: false,
	}
	rtRepo := &mockRefreshTokenRepo{
		getByToken: func(_ context.Context, _ string) (*entity.RefreshToken, error) {
			return expiredToken, nil
		},
	}
	svc := newAuthService(&mockUserRepo{}, rtRepo, newTestJWT(t))

	_, err := svc.RefreshAccessToken(context.Background(), &dto.RefreshTokenRequest{
		RefreshToken: "expired-token",
	})
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	assertServiceErrorCode(t, err, "UNAUTHORIZED")
}

func TestRefreshAccessToken_RevokedToken(t *testing.T) {
	userID := uuid.New()
	revokedToken := &entity.RefreshToken{
		ID:        uuid.New(),
		Token:     "revoked-token",
		UserID:    userID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		IsRevoked: true,
	}
	rtRepo := &mockRefreshTokenRepo{
		getByToken: func(_ context.Context, _ string) (*entity.RefreshToken, error) {
			return revokedToken, nil
		},
	}
	svc := newAuthService(&mockUserRepo{}, rtRepo, newTestJWT(t))

	_, err := svc.RefreshAccessToken(context.Background(), &dto.RefreshTokenRequest{
		RefreshToken: "revoked-token",
	})
	if err == nil {
		t.Fatal("expected error for revoked token, got nil")
	}
	assertServiceErrorCode(t, err, "UNAUTHORIZED")
}

func TestRefreshAccessToken_InactiveUser(t *testing.T) {
	userID := uuid.New()
	_, rtRepo := newValidToken(t, userID)

	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(userID)
			u.IsActive = false
			return u, nil
		},
	}
	svc := newAuthService(userRepo, rtRepo, newTestJWT(t))

	_, err := svc.RefreshAccessToken(context.Background(), &dto.RefreshTokenRequest{
		RefreshToken: "valid-refresh-token",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "FORBIDDEN")
}

// --------------------------------------------------------------------------
// Logout tests
// --------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	revokeCalled := false
	rtRepo := &mockRefreshTokenRepo{
		revoke: func(_ context.Context, _ string) error {
			revokeCalled = true
			return nil
		},
	}
	svc := newAuthService(&mockUserRepo{}, rtRepo, newTestJWT(t))

	err := svc.Logout(context.Background(), &dto.LogoutRequest{RefreshToken: "some-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !revokeCalled {
		t.Error("expected Revoke to be called")
	}
}

func TestLogout_EmptyToken(t *testing.T) {
	svc := newAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, newTestJWT(t))

	err := svc.Logout(context.Background(), &dto.LogoutRequest{RefreshToken: ""})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "BAD_REQUEST")
}

// --------------------------------------------------------------------------
// LogoutAllSessions tests
// --------------------------------------------------------------------------

func TestLogoutAllSessions_Success(t *testing.T) {
	revokeAllCalled := false
	rtRepo := &mockRefreshTokenRepo{
		revokeAllUserTokens: func(_ context.Context, _ uuid.UUID) error {
			revokeAllCalled = true
			return nil
		},
	}
	svc := newAuthService(&mockUserRepo{}, rtRepo, newTestJWT(t))

	err := svc.LogoutAllSessions(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !revokeAllCalled {
		t.Error("expected RevokeAllUserTokens to be called")
	}
}

// --------------------------------------------------------------------------
// GetMe tests
// --------------------------------------------------------------------------

func TestGetMe_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(userID), nil
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	user, err := svc.GetMe(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != userID {
		t.Errorf("expected user id %v, got %v", userID, user.ID)
	}
}

func TestGetMe_InactiveUser(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(userID)
			u.IsActive = false
			return u, nil
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	_, err := svc.GetMe(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "FORBIDDEN")
}

// --------------------------------------------------------------------------
// ChangePassword tests
// --------------------------------------------------------------------------

func TestChangePassword_Success(t *testing.T) {
	userID := uuid.New()
	hashedPw, _ := hashPassword("OldPass123")
	updateCalled := false

	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(userID)
			u.PasswordHash = hashedPw
			return u, nil
		},
		updateUserPassword: func(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) error {
			updateCalled = true
			return nil
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	err := svc.ChangePassword(context.Background(), userID, "OldPass123", "NewPass456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !updateCalled {
		t.Error("expected UpdateUserPassword to be called")
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	userID := uuid.New()
	hashedPw, _ := hashPassword("CorrectOldPass123")

	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(userID)
			u.PasswordHash = hashedPw
			return u, nil
		},
	}
	svc := newAuthService(userRepo, &mockRefreshTokenRepo{}, newTestJWT(t))

	err := svc.ChangePassword(context.Background(), userID, "WrongOldPass123", "NewPass456")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "UNAUTHORIZED")
}

func TestChangePassword_EmptyOldPassword(t *testing.T) {
	svc := newAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, newTestJWT(t))

	err := svc.ChangePassword(context.Background(), uuid.New(), "", "NewPass456")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "BAD_REQUEST")
}

func TestChangePassword_EmptyNewPassword(t *testing.T) {
	svc := newAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, newTestJWT(t))

	err := svc.ChangePassword(context.Background(), uuid.New(), "OldPass123", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "BAD_REQUEST")
}

func TestChangePassword_RevokesAllTokensAfterChange(t *testing.T) {
	userID := uuid.New()
	hashedPw, _ := hashPassword("OldPass123")
	revokeAllCalled := false

	userRepo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(userID)
			u.PasswordHash = hashedPw
			return u, nil
		},
	}
	rtRepo := &mockRefreshTokenRepo{
		revokeAllUserTokens: func(_ context.Context, _ uuid.UUID) error {
			revokeAllCalled = true
			return nil
		},
	}
	svc := newAuthService(userRepo, rtRepo, newTestJWT(t))

	err := svc.ChangePassword(context.Background(), userID, "OldPass123", "NewPass456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !revokeAllCalled {
		t.Error("expected all tokens to be revoked after password change")
	}
}
