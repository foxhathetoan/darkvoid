package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
// Mock: profileService
// --------------------------------------------------------------------------

type mockProfileService struct {
	getMyProfile       func(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	getProfileByUserID func(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	updateMyProfile    func(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*entity.User, error)
	uploadAvatar       func(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error)
	uploadCover        func(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error)
}

func (m *mockProfileService) GetMyProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	if m.getMyProfile != nil {
		return m.getMyProfile(ctx, userID)
	}
	return nil, errors.ErrInternal
}
func (m *mockProfileService) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	if m.getProfileByUserID != nil {
		return m.getProfileByUserID(ctx, userID)
	}
	return nil, errors.ErrInternal
}
func (m *mockProfileService) UpdateMyProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateProfileRequest) (*entity.User, error) {
	if m.updateMyProfile != nil {
		return m.updateMyProfile(ctx, userID, req)
	}
	return nil, errors.ErrInternal
}
func (m *mockProfileService) UploadAvatar(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error) {
	if m.uploadAvatar != nil {
		return m.uploadAvatar(ctx, userID, r, size, contentType, ext)
	}
	return nil, errors.ErrInternal
}
func (m *mockProfileService) UploadCover(ctx context.Context, userID uuid.UUID, r io.Reader, size int64, contentType string, ext string) (*entity.User, error) {
	if m.uploadCover != nil {
		return m.uploadCover(ctx, userID, r, size, contentType, ext)
	}
	return nil, errors.ErrInternal
}

// --------------------------------------------------------------------------
// Test helper
// --------------------------------------------------------------------------

func newProfileHandler(svc profileService) *ProfileHandler {
	return &ProfileHandler{profileService: svc, resolver: nopResolver, storage: nil}
}

func sampleUserFull(id uuid.UUID) *entity.User {
	bio := "Software engineer"
	return &entity.User{
		ID:          id,
		Username:    "johndoe",
		Email:       "john@example.com",
		IsActive:    true,
		DisplayName: "John Doe",
		Bio:         &bio,
		CreatedAt:   time.Now(),
	}
}

// --------------------------------------------------------------------------
// GetMyProfile tests
// --------------------------------------------------------------------------

func TestGetMyProfile_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockProfileService{
		getMyProfile: func(_ context.Context, id uuid.UUID) (*entity.User, error) {
			return sampleUserFull(id), nil
		},
	}
	h := newProfileHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/me", nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.GetMyProfile(w, req)

	assertStatus(t, w, http.StatusOK)

	var resp dto.UserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Username != "johndoe" {
		t.Errorf("expected username johndoe, got %q", resp.Username)
	}
}

func TestGetMyProfile_NotAuthenticated(t *testing.T) {
	h := newProfileHandler(&mockProfileService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	h.GetMyProfile(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestGetMyProfile_NotFound(t *testing.T) {
	userID := uuid.New()
	svc := &mockProfileService{
		getMyProfile: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}
	h := newProfileHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/me", nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.GetMyProfile(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// --------------------------------------------------------------------------
// UpdateMyProfile tests
// --------------------------------------------------------------------------

func TestUpdateMyProfile_Success(t *testing.T) {
	userID := uuid.New()
	bio := "Updated bio"
	svc := &mockProfileService{
		updateMyProfile: func(_ context.Context, id uuid.UUID, req *dto.UpdateProfileRequest) (*entity.User, error) {
			u := sampleUserFull(id)
			u.Bio = req.Bio
			return u, nil
		},
	}
	h := newProfileHandler(svc)

	b, _ := json.Marshal(dto.UpdateProfileRequest{Bio: &bio})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/users/me", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.UpdateMyProfile(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestUpdateMyProfile_NotAuthenticated(t *testing.T) {
	h := newProfileHandler(&mockProfileService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/users/me", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateMyProfile(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestUpdateMyProfile_InvalidJSON(t *testing.T) {
	userID := uuid.New()
	h := newProfileHandler(&mockProfileService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/users/me", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.UpdateMyProfile(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestUpdateMyProfile_ServiceError(t *testing.T) {
	userID := uuid.New()
	svc := &mockProfileService{
		updateMyProfile: func(_ context.Context, _ uuid.UUID, _ *dto.UpdateProfileRequest) (*entity.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}
	h := newProfileHandler(svc)

	b, _ := json.Marshal(dto.UpdateProfileRequest{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/users/me", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(httputil.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	h.UpdateMyProfile(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// --------------------------------------------------------------------------
// GetUserProfile tests
// --------------------------------------------------------------------------

func TestGetUserProfile_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockProfileService{
		getProfileByUserID: func(_ context.Context, id uuid.UUID) (*entity.User, error) {
			return sampleUserFull(id), nil
		},
	}
	h := newProfileHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s/profile", userID), nil)
	req = withChiParam(req, "userKey", userID.String())
	w := httptest.NewRecorder()
	h.GetUserProfile(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestGetUserProfile_InvalidUUID(t *testing.T) {
	h := newProfileHandler(&mockProfileService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/not-a-uuid/profile", nil)
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetUserProfile(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetUserProfile_NotFound(t *testing.T) {
	svc := &mockProfileService{
		getProfileByUserID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}
	h := newProfileHandler(svc)

	userID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s/profile", userID), nil)
	req = withChiParam(req, "userKey", userID.String())
	w := httptest.NewRecorder()
	h.GetUserProfile(w, req)

	assertStatus(t, w, http.StatusNotFound)
}
