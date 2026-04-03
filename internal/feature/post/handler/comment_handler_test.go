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
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	httputil "github.com/jarviisha/darkvoid/internal/http"
	"github.com/jarviisha/darkvoid/internal/pagination"
	pkgerrors "github.com/jarviisha/darkvoid/pkg/errors"
)

// --------------------------------------------------------------------------
// Mock: commentService
// --------------------------------------------------------------------------

type mockCommentService struct {
	createComment func(ctx context.Context, postID, authorID uuid.UUID, parentID *uuid.UUID, content string, mediaKeys []string) (*entity.Comment, error)
	getComments   func(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error)
	getReplies    func(ctx context.Context, commentID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error)
	deleteComment func(ctx context.Context, commentID, userID uuid.UUID) error
}

func (m *mockCommentService) CreateComment(ctx context.Context, postID, authorID uuid.UUID, parentID *uuid.UUID, content string, mediaKeys []string, mentionUserIDs []uuid.UUID) (*entity.Comment, error) {
	if m.createComment != nil {
		return m.createComment(ctx, postID, authorID, parentID, content, mediaKeys)
	}
	return &entity.Comment{ID: uuid.New(), PostID: postID, AuthorID: authorID, Content: content, CreatedAt: time.Now()}, nil
}
func (m *mockCommentService) GetComments(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
	if m.getComments != nil {
		return m.getComments(ctx, postID, viewerID, req)
	}
	return nil, pagination.PaginationResponse{}, nil
}
func (m *mockCommentService) GetReplies(ctx context.Context, commentID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
	if m.getReplies != nil {
		return m.getReplies(ctx, commentID, viewerID, req)
	}
	return nil, pagination.PaginationResponse{}, nil
}
func (m *mockCommentService) DeleteComment(ctx context.Context, commentID, userID uuid.UUID) error {
	if m.deleteComment != nil {
		return m.deleteComment(ctx, commentID, userID)
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers (comment-specific)
// --------------------------------------------------------------------------

func newCommentHandler(svc commentService) *CommentHandler {
	return &CommentHandler{commentService: svc}
}

// --------------------------------------------------------------------------
// CreateComment tests
// --------------------------------------------------------------------------

func TestCreateComment_Success(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	b, _ := json.Marshal(map[string]any{"content": "Great post!"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusCreated)
}

func TestCreateComment_Unauthenticated(t *testing.T) {
	postID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	b, _ := json.Marshal(map[string]any{"content": "hi"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader(b))
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestCreateComment_InvalidPostID(t *testing.T) {
	userID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	b, _ := json.Marshal(map[string]any{"content": "hi"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/bad/comments", bytes.NewReader(b))
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", "bad")
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestCreateComment_InvalidBody(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader([]byte("not-json")))
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestCreateComment_WithParentID(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	var capturedParentID *uuid.UUID
	svc := &mockCommentService{
		createComment: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, pid *uuid.UUID, _ string, _ []string) (*entity.Comment, error) {
			capturedParentID = pid
			return &entity.Comment{ID: uuid.New(), PostID: postID, Content: "Reply"}, nil
		},
	}
	h := newCommentHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": "Reply!", "parent_id": parentIDStr})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusCreated)
	if capturedParentID == nil || *capturedParentID != parentID {
		t.Errorf("expected parent ID %v, got %v", parentID, capturedParentID)
	}
}

func TestCreateComment_InvalidParentID(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	invalidParentID := "not-a-uuid"
	h := newCommentHandler(&mockCommentService{})

	b, _ := json.Marshal(map[string]any{"content": "Reply!", "parent_id": invalidParentID})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestCreateComment_PostNotFound(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockCommentService{
		createComment: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *uuid.UUID, _ string, _ []string) (*entity.Comment, error) {
			return nil, post.ErrPostNotFound
		},
	}
	h := newCommentHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": "hi"})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusNotFound)
	assertErrCode(t, w, "POST_NOT_FOUND")
}

func TestCreateComment_EmptyContent_ServiceError(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	svc := &mockCommentService{
		createComment: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *uuid.UUID, _ string, _ []string) (*entity.Comment, error) {
			return nil, pkgerrors.NewBadRequestError("comment content is required")
		},
	}
	h := newCommentHandler(svc)

	b, _ := json.Marshal(map[string]any{"content": "   "})
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/posts/"+postID.String()+"/comments", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r = withAuth(r, userID)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.CreateComment(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

// --------------------------------------------------------------------------
// GetComments tests
// --------------------------------------------------------------------------

func TestGetComments_Success(t *testing.T) {
	postID := uuid.New()
	svc := &mockCommentService{
		getComments: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
			return []*entity.Comment{
				{ID: uuid.New(), PostID: postID, Content: "c1", CreatedAt: time.Now()},
			}, pagination.PaginationResponse{Total: 1}, nil
		},
	}
	h := newCommentHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/"+postID.String()+"/comments", nil)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.GetComments(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetComments_InvalidPostID(t *testing.T) {
	h := newCommentHandler(&mockCommentService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/bad/comments", nil)
	r = withChiParam(r, "postID", "bad")
	w := httptest.NewRecorder()

	h.GetComments(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetComments_PostNotFound(t *testing.T) {
	postID := uuid.New()
	svc := &mockCommentService{
		getComments: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
			return nil, pagination.PaginationResponse{}, post.ErrPostNotFound
		},
	}
	h := newCommentHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/"+postID.String()+"/comments", nil)
	r = withChiParam(r, "postID", postID.String())
	w := httptest.NewRecorder()

	h.GetComments(w, r)
	assertStatus(t, w, http.StatusNotFound)
	assertErrCode(t, w, "POST_NOT_FOUND")
}

// --------------------------------------------------------------------------
// GetReplies tests
// --------------------------------------------------------------------------

func TestGetReplies_Success(t *testing.T) {
	postID := uuid.New()
	commentID := uuid.New()
	svc := &mockCommentService{
		getReplies: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
			return []*entity.Comment{
				{ID: uuid.New(), PostID: postID, ParentID: &commentID, Content: "reply1", CreatedAt: time.Now()},
			}, pagination.PaginationResponse{Total: 1}, nil
		},
	}
	h := newCommentHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/"+postID.String()+"/comments/"+commentID.String()+"/replies", nil)
	r = withChiParam(r, "commentID", commentID.String())
	w := httptest.NewRecorder()

	h.GetReplies(w, r)
	assertStatus(t, w, http.StatusOK)
}

func TestGetReplies_InvalidCommentID(t *testing.T) {
	h := newCommentHandler(&mockCommentService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/posts/x/comments/bad/replies", nil)
	r = withChiParam(r, "commentID", "bad")
	w := httptest.NewRecorder()

	h.GetReplies(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetReplies_CommentNotFound(t *testing.T) {
	commentID := uuid.New()
	svc := &mockCommentService{
		getReplies: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
			return nil, pagination.PaginationResponse{}, post.ErrCommentNotFound
		},
	}
	h := newCommentHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/x/comments/"+commentID.String()+"/replies", nil)
	r = withChiParam(r, "commentID", commentID.String())
	w := httptest.NewRecorder()

	h.GetReplies(w, r)
	assertStatus(t, w, http.StatusNotFound)
	assertErrCode(t, w, "COMMENT_NOT_FOUND")
}

// --------------------------------------------------------------------------
// DeleteComment tests
// --------------------------------------------------------------------------

func TestDeleteComment_Success(t *testing.T) {
	userID := uuid.New()
	commentID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/comments/"+commentID.String(), nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "commentID", commentID.String())
	w := httptest.NewRecorder()

	h.DeleteComment(w, r)
	assertStatus(t, w, http.StatusNoContent)
}

func TestDeleteComment_Unauthenticated(t *testing.T) {
	commentID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/comments/"+commentID.String(), nil)
	r = withChiParam(r, "commentID", commentID.String())
	w := httptest.NewRecorder()

	h.DeleteComment(w, r)
	assertStatus(t, w, http.StatusUnauthorized)
}

func TestDeleteComment_InvalidCommentID(t *testing.T) {
	userID := uuid.New()
	h := newCommentHandler(&mockCommentService{})

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/comments/bad", nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "commentID", "bad")
	w := httptest.NewRecorder()

	h.DeleteComment(w, r)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestDeleteComment_Forbidden(t *testing.T) {
	userID := uuid.New()
	commentID := uuid.New()
	svc := &mockCommentService{
		deleteComment: func(_ context.Context, _, _ uuid.UUID) error {
			return post.ErrForbidden
		},
	}
	h := newCommentHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/comments/"+commentID.String(), nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "commentID", commentID.String())
	w := httptest.NewRecorder()

	h.DeleteComment(w, r)
	assertStatus(t, w, http.StatusForbidden)
	assertErrCode(t, w, "FORBIDDEN")
}

func TestDeleteComment_NotFound(t *testing.T) {
	userID := uuid.New()
	commentID := uuid.New()
	svc := &mockCommentService{
		deleteComment: func(_ context.Context, _, _ uuid.UUID) error {
			return post.ErrCommentNotFound
		},
	}
	h := newCommentHandler(svc)

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/comments/"+commentID.String(), nil)
	r = withAuth(r, userID)
	r = withChiParam(r, "commentID", commentID.String())
	w := httptest.NewRecorder()

	h.DeleteComment(w, r)
	assertStatus(t, w, http.StatusNotFound)
	assertErrCode(t, w, "COMMENT_NOT_FOUND")
}

// Ensure httputil import is used (it is, via withAuth which calls httputil.WithUserID)
var _ = httputil.WithUserID
