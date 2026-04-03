package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/feed"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	pkgerrors "github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

type mockFeedService struct {
	getFeed     func(ctx context.Context, userID uuid.UUID, cursor *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error)
	getDiscover func(ctx context.Context, viewerID *uuid.UUID, cursor *feed.DiscoverCursor, limit int32) ([]*feedentity.Post, *feed.DiscoverCursor, error)
}

func (m *mockFeedService) GetFeed(ctx context.Context, userID uuid.UUID, cursor *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error) {
	if m.getFeed != nil {
		return m.getFeed(ctx, userID, cursor)
	}
	return nil, nil, nil
}

func (m *mockFeedService) GetDiscover(ctx context.Context, viewerID *uuid.UUID, cursor *feed.DiscoverCursor, limit int32) ([]*feedentity.Post, *feed.DiscoverCursor, error) {
	if m.getDiscover != nil {
		return m.getDiscover(ctx, viewerID, cursor, limit)
	}
	return nil, nil, nil
}

func samplePost(authorID uuid.UUID) *feedentity.Post {
	return &feedentity.Post{
		ID:         uuid.New(),
		AuthorID:   authorID,
		Content:    "Hello world",
		Visibility: "public",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func sampleFeedItem(authorID uuid.UUID) *feedentity.FeedItem {
	return &feedentity.FeedItem{
		Post:   samplePost(authorID),
		Score:  42.0,
		Source: feedentity.SourceFollowing,
	}
}

func nopStore() storage.Storage { return storage.NewNop("") }

func newFeedHandler(svc feedService) *FeedHandler {
	return &FeedHandler{feedService: svc, store: nopStore()}
}

func withAuth(r *http.Request, userID uuid.UUID) *http.Request {
	return r.WithContext(httputil.WithUserID(r.Context(), userID))
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("expected status %d, got %d (body: %s)", want, w.Code, w.Body.String())
	}
}

// GetFeed tests

func TestGetFeed_Success(t *testing.T) {
	userID := uuid.New()
	svc := &mockFeedService{
		getFeed: func(_ context.Context, _ uuid.UUID, _ *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error) {
			return []*feedentity.FeedItem{sampleFeedItem(uuid.New())}, nil, nil
		},
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/feed", nil)
	r := withAuth(req, userID)
	w := httptest.NewRecorder()
	newFeedHandler(svc).GetFeed(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetFeed_WithNextCursor(t *testing.T) {
	userID := uuid.New()
	next := &feed.FollowingCursor{CreatedAt: time.Now(), PostID: uuid.New().String()}
	svc := &mockFeedService{
		getFeed: func(_ context.Context, _ uuid.UUID, _ *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error) {
			return []*feedentity.FeedItem{sampleFeedItem(uuid.New())}, next, nil
		},
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/feed", nil)
	r := withAuth(req, userID)
	w := httptest.NewRecorder()
	newFeedHandler(svc).GetFeed(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetFeed_InvalidCursor(t *testing.T) {
	userID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/feed?cursor=not-valid!!!", nil)
	r := withAuth(req, userID)
	w := httptest.NewRecorder()
	newFeedHandler(&mockFeedService{}).GetFeed(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetFeed_Unauthenticated(t *testing.T) {
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/feed", nil)
	w := httptest.NewRecorder()
	newFeedHandler(&mockFeedService{}).GetFeed(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestGetFeed_ServiceError(t *testing.T) {
	userID := uuid.New()
	svc := &mockFeedService{
		getFeed: func(_ context.Context, _ uuid.UUID, _ *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error) {
			return nil, nil, pkgerrors.NewInternalError(errors.New("db error"))
		},
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/feed", nil)
	r := withAuth(req, userID)
	w := httptest.NewRecorder()
	newFeedHandler(svc).GetFeed(w, r)
	assertStatus(t, w, http.StatusInternalServerError)
}

// GetDiscover tests

func TestGetDiscover_Success(t *testing.T) {
	svc := &mockFeedService{
		getDiscover: func(_ context.Context, _ *uuid.UUID, _ *feed.DiscoverCursor, _ int32) ([]*feedentity.Post, *feed.DiscoverCursor, error) {
			return []*feedentity.Post{samplePost(uuid.New())}, nil, nil
		},
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/discover", nil)
	w := httptest.NewRecorder()
	newFeedHandler(svc).GetDiscover(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetDiscover_WithNextCursor(t *testing.T) {
	next := &feed.DiscoverCursor{CreatedAt: time.Now(), PostID: uuid.New().String()}
	svc := &mockFeedService{
		getDiscover: func(_ context.Context, _ *uuid.UUID, _ *feed.DiscoverCursor, _ int32) ([]*feedentity.Post, *feed.DiscoverCursor, error) {
			return []*feedentity.Post{samplePost(uuid.New())}, next, nil
		},
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/discover", nil)
	w := httptest.NewRecorder()
	newFeedHandler(svc).GetDiscover(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetDiscover_InvalidCursor(t *testing.T) {
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/discover?cursor=not-valid!!!", nil)
	w := httptest.NewRecorder()
	newFeedHandler(&mockFeedService{}).GetDiscover(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetDiscover_NoAuthStillWorks(t *testing.T) {
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/discover", nil)
	w := httptest.NewRecorder()
	newFeedHandler(&mockFeedService{}).GetDiscover(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetDiscover_ServiceError(t *testing.T) {
	svc := &mockFeedService{
		getDiscover: func(_ context.Context, _ *uuid.UUID, _ *feed.DiscoverCursor, _ int32) ([]*feedentity.Post, *feed.DiscoverCursor, error) {
			return nil, nil, pkgerrors.NewInternalError(errors.New("db error"))
		},
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/discover", nil)
	w := httptest.NewRecorder()
	newFeedHandler(svc).GetDiscover(w, r)
	assertStatus(t, w, http.StatusInternalServerError)
}
