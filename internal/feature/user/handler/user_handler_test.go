package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// --------------------------------------------------------------------------
// Mock: userService
// --------------------------------------------------------------------------

type mockUserService struct {
	getUserByID    func(ctx context.Context, id uuid.UUID) (*entity.User, error)
	updateUser     func(ctx context.Context, id uuid.UUID, req *dto.UpdateUserRequest, updatedBy *uuid.UUID) (*entity.User, error)
	deactivateUser func(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error
}

func (m *mockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, id)
	}
	return nil, errors.ErrInternal
}
func (m *mockUserService) UpdateUser(ctx context.Context, id uuid.UUID, req *dto.UpdateUserRequest, updatedBy *uuid.UUID) (*entity.User, error) {
	if m.updateUser != nil {
		return m.updateUser(ctx, id, req, updatedBy)
	}
	return nil, errors.ErrInternal
}
func (m *mockUserService) DeactivateUser(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error {
	if m.deactivateUser != nil {
		return m.deactivateUser(ctx, id, updatedBy)
	}
	return nil
}

// --------------------------------------------------------------------------
// Mock: userResolver (shared across test files)
// --------------------------------------------------------------------------

type mockUserResolver struct {
	getUserByUsername func(ctx context.Context, username string) (*entity.User, error)
}

func (m *mockUserResolver) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	if m.getUserByUsername != nil {
		return m.getUserByUsername(ctx, username)
	}
	return nil, errors.NewNotFoundError("user not found")
}

// nopResolver is a no-op resolver for tests that only use UUID lookups.
var nopResolver = &mockUserResolver{}

// --------------------------------------------------------------------------
// Test helpers
// --------------------------------------------------------------------------

func newUserHandler(svc userService) *UserHandler {
	return &UserHandler{userService: svc, storage: nil}
}

func sampleUser(id uuid.UUID) *entity.User {
	return &entity.User{
		ID:          id,
		Username:    "johndoe",
		Email:       "john@example.com",
		IsActive:    true,
		DisplayName: "John Doe",
		CreatedAt:   time.Now(),
	}
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("expected status %d, got %d (body: %s)", expected, w.Code, w.Body.String())
	}
}

func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, code string) {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Error.Code != code {
		t.Errorf("expected error code %q, got %q", code, resp.Error.Code)
	}
}

func withChiParam(r *http.Request, _, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("userKey", value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

func withUserID(r *http.Request, id uuid.UUID) *http.Request {
	return r.WithContext(httputil.WithUserID(r.Context(), id))
}

func putJSON(t *testing.T, url string, body interface{}) *http.Request {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

// --------------------------------------------------------------------------
// GetUser tests
// --------------------------------------------------------------------------

func TestGetUser_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockUserService{
		getUserByID: func(_ context.Context, id uuid.UUID) (*entity.User, error) {
			return sampleUser(id), nil
		},
	}
	h := newUserHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s", userID), nil)
	req = withChiParam(req, "userKey", userID.String())
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestGetUser_InvalidUUID(t *testing.T) {
	h := newUserHandler(&mockUserService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/not-a-uuid", nil)
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetUser_NotFound(t *testing.T) {
	svc := &mockUserService{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}
	h := newUserHandler(svc)

	userID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s", userID), nil)
	req = withChiParam(req, "userKey", userID.String())
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// --------------------------------------------------------------------------
// UpdateUser tests
// --------------------------------------------------------------------------

func TestUpdateUser_Success(t *testing.T) {
	userID := uuid.New()
	newEmail := "new@example.com"
	svc := &mockUserService{
		updateUser: func(_ context.Context, id uuid.UUID, _ *dto.UpdateUserRequest, _ *uuid.UUID) (*entity.User, error) {
			u := sampleUser(id)
			u.Email = newEmail
			return u, nil
		},
	}
	h := newUserHandler(svc)

	req := putJSON(t, fmt.Sprintf("/users/%s", userID), map[string]string{"email": newEmail})
	req = withChiParam(req, "userKey", userID.String())
	req = withUserID(req, userID)
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestUpdateUser_InvalidUUID(t *testing.T) {
	h := newUserHandler(&mockUserService{})

	req := putJSON(t, "/users/not-a-uuid", map[string]string{"email": "x@example.com"})
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestUpdateUser_InvalidJSON(t *testing.T) {
	h := newUserHandler(&mockUserService{})

	userID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, fmt.Sprintf("/users/%s", userID), bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParam(req, "userKey", userID.String())
	req = withUserID(req, userID)
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestUpdateUser_NotFound(t *testing.T) {
	svc := &mockUserService{
		updateUser: func(_ context.Context, _ uuid.UUID, _ *dto.UpdateUserRequest, _ *uuid.UUID) (*entity.User, error) {
			return nil, errors.NewNotFoundError("user not found")
		},
	}
	h := newUserHandler(svc)

	userID := uuid.New()
	req := putJSON(t, fmt.Sprintf("/users/%s", userID), map[string]string{"email": "x@example.com"})
	req = withChiParam(req, "userKey", userID.String())
	req = withUserID(req, userID)
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

func TestUpdateUser_ConflictEmail(t *testing.T) {
	svc := &mockUserService{
		updateUser: func(_ context.Context, _ uuid.UUID, _ *dto.UpdateUserRequest, _ *uuid.UUID) (*entity.User, error) {
			return nil, errors.NewConflictError("email already exists")
		},
	}
	h := newUserHandler(svc)

	userID := uuid.New()
	req := putJSON(t, fmt.Sprintf("/users/%s", userID), map[string]string{"email": "taken@example.com"})
	req = withChiParam(req, "userKey", userID.String())
	req = withUserID(req, userID)
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assertStatus(t, w, http.StatusConflict)
}

// --------------------------------------------------------------------------
// DeactivateUser tests
// --------------------------------------------------------------------------

func TestDeactivateUser_Success(t *testing.T) {
	svc := &mockUserService{
		deactivateUser: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
			return nil
		},
	}
	h := newUserHandler(svc)

	userID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/users/%s", userID), nil)
	req = withChiParam(req, "userKey", userID.String())
	req = withUserID(req, userID)
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestDeactivateUser_InvalidUUID(t *testing.T) {
	h := newUserHandler(&mockUserService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/users/not-a-uuid", nil)
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestDeactivateUser_NotFound(t *testing.T) {
	svc := &mockUserService{
		deactivateUser: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
			return errors.NewNotFoundError("user not found")
		},
	}
	h := newUserHandler(svc)

	userID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/users/%s", userID), nil)
	req = withChiParam(req, "userKey", userID.String())
	req = withUserID(req, userID)
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)

	assertStatus(t, w, http.StatusNotFound)
}
