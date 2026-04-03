package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
)

// --------------------------------------------------------------------------
// Mock: likeService
// --------------------------------------------------------------------------

type mockLikeService struct {
	toggle func(ctx context.Context, userID, postID uuid.UUID) (bool, error)
}

func (m *mockLikeService) Toggle(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	if m.toggle != nil {
		return m.toggle(ctx, userID, postID)
	}
	return true, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func newLikeHandler(svc likeService) *LikeHandler {
	return &LikeHandler{likeService: svc}
}

func assertLiked(t *testing.T, w *httptest.ResponseRecorder, want bool) {
	t.Helper()
	var body map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if got := body["liked"]; got != want {
		t.Errorf("liked = %v, want %v", got, want)
	}
}

// --------------------------------------------------------------------------
// ToggleLike tests
// --------------------------------------------------------------------------

func TestToggleLike_Like(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockLikeService{
		toggle: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	h := newLikeHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/like", nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.ToggleLike(w, r)
	assertStatus(t, w, http.StatusOK)
	assertLiked(t, w, true)
}

func TestToggleLike_Unlike(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockLikeService{
		toggle: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}
	h := newLikeHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/like", nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.ToggleLike(w, r)
	assertStatus(t, w, http.StatusOK)
	assertLiked(t, w, false)
}

func TestToggleLike_Unauthenticated(t *testing.T) {
	postID := uuid.New()
	h := newLikeHandler(&mockLikeService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/like", nil)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.ToggleLike(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestToggleLike_InvalidPostID(t *testing.T) {
	userID := uuid.New()
	h := newLikeHandler(&mockLikeService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/bad/like", nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", "bad")
	w := httptest.NewRecorder()

	h.ToggleLike(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestToggleLike_PostNotFound(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockLikeService{
		toggle: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, post.ErrPostNotFound
		},
	}
	h := newLikeHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/like", nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.ToggleLike(w, r)
	assertStatus(t, w, http.StatusNotFound)
	assertErrCode(t, w, "POST_NOT_FOUND")
}

func TestToggleLike_SelfLike(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockLikeService{
		toggle: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, post.ErrSelfLike
		},
	}
	h := newLikeHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/like", nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.ToggleLike(w, r)
	assertStatus(t, w, http.StatusBadRequest)
	assertErrCode(t, w, "SELF_LIKE")
}
