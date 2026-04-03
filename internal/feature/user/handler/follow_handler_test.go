package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// --------------------------------------------------------------------------
// Mock: followService
// --------------------------------------------------------------------------

type mockFollowService struct {
	follow       func(ctx context.Context, followerID, followeeID uuid.UUID) error
	unfollow     func(ctx context.Context, followerID, followeeID uuid.UUID) error
	getFollowers func(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error)
	getFollowing func(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error)
}

func (m *mockFollowService) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if m.follow != nil {
		return m.follow(ctx, followerID, followeeID)
	}
	return nil
}
func (m *mockFollowService) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if m.unfollow != nil {
		return m.unfollow(ctx, followerID, followeeID)
	}
	return nil
}
func (m *mockFollowService) GetFollowers(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
	if m.getFollowers != nil {
		return m.getFollowers(ctx, targetID, req)
	}
	return nil, pagination.PaginationResponse{}, errors.ErrInternal
}
func (m *mockFollowService) GetFollowing(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
	if m.getFollowing != nil {
		return m.getFollowing(ctx, targetID, req)
	}
	return nil, pagination.PaginationResponse{}, errors.ErrInternal
}

// --------------------------------------------------------------------------
// Test helper
// --------------------------------------------------------------------------

func newFollowHandler(svc followService) *FollowHandler {
	return &FollowHandler{followService: svc, resolver: nopResolver}
}

func sampleFollow(followerID, followeeID uuid.UUID) *entity.Follow {
	return &entity.Follow{
		FollowerID: followerID,
		FolloweeID: followeeID,
		CreatedAt:  time.Now(),
	}
}

// --------------------------------------------------------------------------
// Follow tests
// --------------------------------------------------------------------------

func TestFollow_Success(t *testing.T) {
	followerID := uuid.New()
	followeeID := uuid.New()
	followCalled := false
	svc := &mockFollowService{
		follow: func(_ context.Context, fwr, fwe uuid.UUID) error {
			followCalled = true
			if fwr != followerID {
				t.Errorf("expected followerID %v, got %v", followerID, fwr)
			}
			if fwe != followeeID {
				t.Errorf("expected followeeID %v, got %v", followeeID, fwe)
			}
			return nil
		},
	}
	h := newFollowHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("/users/%s/follow", followeeID), nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), followerID))
	req = withChiParam(req, "userKey", followeeID.String())
	w := httptest.NewRecorder()
	h.Follow(w, req)

	assertStatus(t, w, http.StatusNoContent)
	if !followCalled {
		t.Error("expected Follow to be called on service")
	}
}

func TestFollow_NotAuthenticated(t *testing.T) {
	h := newFollowHandler(&mockFollowService{})

	followeeID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("/users/%s/follow", followeeID), nil)
	req = withChiParam(req, "userKey", followeeID.String())
	w := httptest.NewRecorder()
	h.Follow(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestFollow_InvalidUserID(t *testing.T) {
	followerID := uuid.New()
	h := newFollowHandler(&mockFollowService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/users/not-a-uuid/follow", nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), followerID))
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Follow(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestFollow_ServiceError(t *testing.T) {
	followerID := uuid.New()
	followeeID := uuid.New()
	svc := &mockFollowService{
		follow: func(_ context.Context, _, _ uuid.UUID) error {
			return errors.NewBadRequestError("cannot follow yourself")
		},
	}
	h := newFollowHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("/users/%s/follow", followeeID), nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), followerID))
	req = withChiParam(req, "userKey", followeeID.String())
	w := httptest.NewRecorder()
	h.Follow(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// --------------------------------------------------------------------------
// Unfollow tests
// --------------------------------------------------------------------------

func TestUnfollow_Success(t *testing.T) {
	followerID := uuid.New()
	followeeID := uuid.New()
	svc := &mockFollowService{
		unfollow: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	h := newFollowHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/users/%s/follow", followeeID), nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), followerID))
	req = withChiParam(req, "userKey", followeeID.String())
	w := httptest.NewRecorder()
	h.Unfollow(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

func TestUnfollow_NotAuthenticated(t *testing.T) {
	h := newFollowHandler(&mockFollowService{})

	followeeID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/users/%s/follow", followeeID), nil)
	req = withChiParam(req, "userKey", followeeID.String())
	w := httptest.NewRecorder()
	h.Unfollow(w, req)

	assertStatus(t, w, http.StatusUnauthorized)
	assertErrorCode(t, w, "UNAUTHORIZED")
}

func TestUnfollow_InvalidUserID(t *testing.T) {
	followerID := uuid.New()
	h := newFollowHandler(&mockFollowService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/users/not-a-uuid/follow", nil)
	req = req.WithContext(httputil.WithUserID(req.Context(), followerID))
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Unfollow(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// --------------------------------------------------------------------------
// GetFollowers tests
// --------------------------------------------------------------------------

func TestGetFollowers_Success(t *testing.T) {
	targetID := uuid.New()
	follows := []*entity.Follow{
		sampleFollow(uuid.New(), targetID),
		sampleFollow(uuid.New(), targetID),
	}
	svc := &mockFollowService{
		getFollowers: func(_ context.Context, _ uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
			return follows, pagination.PaginationResponse{Total: 2, Limit: 20, Offset: 0}, nil
		},
	}
	h := newFollowHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s/followers", targetID), nil)
	req = withChiParam(req, "userKey", targetID.String())
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestGetFollowers_InvalidUUID(t *testing.T) {
	h := newFollowHandler(&mockFollowService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/not-a-uuid/followers", nil)
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetFollowers_ServiceError(t *testing.T) {
	svc := &mockFollowService{
		getFollowers: func(_ context.Context, _ uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
			return nil, pagination.PaginationResponse{}, errors.ErrInternal
		},
	}
	h := newFollowHandler(svc)

	targetID := uuid.New()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s/followers", targetID), nil)
	req = withChiParam(req, "userKey", targetID.String())
	w := httptest.NewRecorder()
	h.GetFollowers(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// --------------------------------------------------------------------------
// GetFollowing tests
// --------------------------------------------------------------------------

func TestGetFollowing_Success(t *testing.T) {
	targetID := uuid.New()
	follows := []*entity.Follow{
		sampleFollow(targetID, uuid.New()),
	}
	svc := &mockFollowService{
		getFollowing: func(_ context.Context, _ uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
			return follows, pagination.PaginationResponse{Total: 1, Limit: 20, Offset: 0}, nil
		},
	}
	h := newFollowHandler(svc)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/users/%s/following", targetID), nil)
	req = withChiParam(req, "userKey", targetID.String())
	w := httptest.NewRecorder()
	h.GetFollowing(w, req)

	assertStatus(t, w, http.StatusOK)
}

func TestGetFollowing_InvalidUUID(t *testing.T) {
	h := newFollowHandler(&mockFollowService{})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/users/not-a-uuid/following", nil)
	req = withChiParam(req, "userKey", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetFollowing(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}
