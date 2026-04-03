package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// --------------------------------------------------------------------------
// Mock: postService
// --------------------------------------------------------------------------

type mockPostService struct {
	createPost   func(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility, mediaKeys []string) (*entity.Post, error)
	getPost      func(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID) (*entity.Post, error)
	updatePost   func(ctx context.Context, postID, userID uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error)
	deletePost   func(ctx context.Context, postID, userID uuid.UUID) error
	getUserPosts func(ctx context.Context, authorID uuid.UUID, viewerID *uuid.UUID, cursor *post.UserPostCursor, visibility string, limit int32) ([]*entity.Post, *post.UserPostCursor, error)
}

func (m *mockPostService) CreatePost(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility, mediaKeys []string, mentionUserIDs []uuid.UUID, tags []string) (*entity.Post, error) {
	if m.createPost != nil {
		return m.createPost(ctx, authorID, content, visibility, mediaKeys)
	}
	return samplePost(authorID), nil
}
func (m *mockPostService) GetPost(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID) (*entity.Post, error) {
	if m.getPost != nil {
		return m.getPost(ctx, postID, viewerID)
	}
	return samplePost(uuid.New()), nil
}
func (m *mockPostService) UpdatePost(ctx context.Context, postID, userID uuid.UUID, content string, visibility entity.Visibility, mentionUserIDs []uuid.UUID, tags []string) (*entity.Post, error) {
	if m.updatePost != nil {
		return m.updatePost(ctx, postID, userID, content, visibility)
	}
	return samplePost(userID), nil
}
func (m *mockPostService) DeletePost(ctx context.Context, postID, userID uuid.UUID) error {
	if m.deletePost != nil {
		return m.deletePost(ctx, postID, userID)
	}
	return nil
}
func (m *mockPostService) GetUserPosts(ctx context.Context, authorID uuid.UUID, viewerID *uuid.UUID, cursor *post.UserPostCursor, visibility string, limit int32) ([]*entity.Post, *post.UserPostCursor, error) {
	if m.getUserPosts != nil {
		return m.getUserPosts(ctx, authorID, viewerID, cursor, visibility, limit)
	}
	return nil, nil, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func samplePost(authorID uuid.UUID) *entity.Post {
	return &entity.Post{
		ID:         uuid.New(),
		AuthorID:   authorID,
		Content:    "Hello world",
		Visibility: entity.VisibilityPublic,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func nopStore() storage.Storage {
	return storage.NewNop("")
}

func newPostHandler(svc postService) *PostHandler {
	return &PostHandler{
		postService: svc,
		store:       nopStore(),
	}
}

func withAuth(r *http.Request, userID uuid.UUID) *http.Request {
	return r.WithContext(httputil.WithUserID(r.Context(), userID))
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("expected status %d, got %d (body: %s)", want, w.Code, w.Body.String())
	}
}

type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeErrResp(t *testing.T, w *httptest.ResponseRecorder) errResp {
	t.Helper()
	var e errResp
	if err := json.NewDecoder(w.Body).Decode(&e); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return e
}

func assertErrCode(t *testing.T, w *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	e := decodeErrResp(t, w)
	if e.Error.Code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, e.Error.Code)
	}
}

// --------------------------------------------------------------------------
// CreatePost tests
// --------------------------------------------------------------------------

func TestCreatePost_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockPostService{}
	h := newPostHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": "Hello", "visibility": "public"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	w := httptest.NewRecorder()

	h.CreatePost(w, r)
	assertStatus(t, w, http.StatusCreated)
}

func TestCreatePost_Unauthenticated(t *testing.T) {
	h := newPostHandler(&mockPostService{})

	b, _ := json.Marshal(map[string]any{"content": "Hello"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts", bytes.NewReader(b))
	w := httptest.NewRecorder()

	h.CreatePost(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestCreatePost_InvalidBody(t *testing.T) {
	userID := uuid.New()
	h := newPostHandler(&mockPostService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts", bytes.NewReader([]byte("not-json")))
	r = withAuth(r, userID)
	w := httptest.NewRecorder()

	h.CreatePost(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestCreatePost_ServiceError(t *testing.T) {
	userID := uuid.New()
	svc := &mockPostService{
		createPost: func(_ context.Context, _ uuid.UUID, _ string, _ entity.Visibility, _ []string) (*entity.Post, error) {
			return nil, post.ErrEmptyContent
		},
	}
	h := newPostHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": ""})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	w := httptest.NewRecorder()

	h.CreatePost(w, r)
	assertStatus(t, w, http.StatusBadRequest)
	assertErrCode(t, w, "EMPTY_CONTENT")
}

func TestCreatePost_DefaultVisibility(t *testing.T) {
	userID := uuid.New()
	var capturedVisibility entity.Visibility
	svc := &mockPostService{
		createPost: func(_ context.Context, _ uuid.UUID, _ string, vis entity.Visibility, _ []string) (*entity.Post, error) {
			capturedVisibility = vis
			return samplePost(userID), nil
		},
	}
	h := newPostHandler(svc)

	// omit visibility — should default to public
	b, _ := json.Marshal(map[string]any{"content": "hi"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	w := httptest.NewRecorder()

	h.CreatePost(w, r)
	assertStatus(t, w, http.StatusCreated)
	if capturedVisibility != entity.VisibilityPublic {
		t.Errorf("expected default visibility %q, got %q", entity.VisibilityPublic, capturedVisibility)
	}
}

// --------------------------------------------------------------------------
// GetPost tests
// --------------------------------------------------------------------------

func TestGetPost_Success(t *testing.T) {
	postID := uuid.New()
	svc := &mockPostService{
		getPost: func(_ context.Context, id uuid.UUID, _ *uuid.UUID) (*entity.Post, error) {
			return samplePost(uuid.New()), nil
		},
	}
	h := newPostHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/"+postID.String(), nil)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.GetPost(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetPost_InvalidID(t *testing.T) {
	h := newPostHandler(&mockPostService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/not-a-uuid", nil)
	r = withChiParam(r, "postID", "not-a-uuid")
	w := httptest.NewRecorder()

	h.GetPost(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetPost_NotFound(t *testing.T) {
	postID := uuid.New()
	svc := &mockPostService{
		getPost: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (*entity.Post, error) {
			return nil, post.ErrPostNotFound
		},
	}
	h := newPostHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/"+postID.String(), nil)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.GetPost(w, r)
	assertStatus(t, w, http.StatusNotFound)
	assertErrCode(t, w, "POST_NOT_FOUND")
}

// --------------------------------------------------------------------------
// UpdatePost tests
// --------------------------------------------------------------------------

func TestUpdatePost_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockPostService{}
	h := newPostHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": "Updated", "visibility": "public"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/posts/"+postID.String(), bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.UpdatePost(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestUpdatePost_Unauthenticated(t *testing.T) {
	postID := uuid.New()
	h := newPostHandler(&mockPostService{})

	b, _ := json.Marshal(map[string]any{"content": "x"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/posts/"+postID.String(), bytes.NewReader(b))
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.UpdatePost(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestUpdatePost_InvalidPostID(t *testing.T) {
	userID := uuid.New()
	h := newPostHandler(&mockPostService{})

	b, _ := json.Marshal(map[string]any{"content": "x"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/posts/bad", bytes.NewReader(b))
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", "bad")
	w := httptest.NewRecorder()

	h.UpdatePost(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestUpdatePost_Forbidden(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockPostService{
		updatePost: func(_ context.Context, _, _ uuid.UUID, _ string, _ entity.Visibility) (*entity.Post, error) {
			return nil, post.ErrForbidden
		},
	}
	h := newPostHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": "x"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, "/posts/"+postID.String(), bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.UpdatePost(w, r)
	assertStatus(t, w, http.StatusForbidden)
	assertErrCode(t, w, "FORBIDDEN")
}

// --------------------------------------------------------------------------
// DeletePost tests
// --------------------------------------------------------------------------

func TestDeletePost_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	h := newPostHandler(&mockPostService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/posts/"+postID.String(), nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.DeletePost(w, r)
	assertStatus(t, w, http.StatusNoContent)
}

func TestDeletePost_Unauthenticated(t *testing.T) {
	postID := uuid.New()
	h := newPostHandler(&mockPostService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/posts/"+postID.String(), nil)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.DeletePost(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestDeletePost_NotFound(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockPostService{
		deletePost: func(_ context.Context, _, _ uuid.UUID) error {
			return post.ErrPostNotFound
		},
	}
	h := newPostHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/posts/"+postID.String(), nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.DeletePost(w, r)
	assertStatus(t, w, http.StatusNotFound)
}

// --------------------------------------------------------------------------
// GetUserPosts tests
// --------------------------------------------------------------------------

func TestGetUserPosts_Success(t *testing.T) {
	authorID := uuid.New()
	svc := &mockPostService{
		getUserPosts: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ *post.UserPostCursor, _ string, _ int32) ([]*entity.Post, *post.UserPostCursor, error) {
			return []*entity.Post{samplePost(authorID)}, nil, nil
		},
	}
	h := newPostHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/"+authorID.String()+"/posts", nil)
	r = withChiParam(r, "userID", authorID.String())
	w := httptest.NewRecorder()

	h.GetUserPosts(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetUserPosts_InvalidUserID(t *testing.T) {
	h := newPostHandler(&mockPostService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/bad/posts", nil)
	r = withChiParam(r, "userID", "bad")
	w := httptest.NewRecorder()

	h.GetUserPosts(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}
