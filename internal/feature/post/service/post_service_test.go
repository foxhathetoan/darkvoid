package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	pkgerrors "github.com/jarviisha/darkvoid/pkg/errors"
)

// --------------------------------------------------------------------------
// Transaction mocks
// --------------------------------------------------------------------------

// mockTx is a no-op pgx.Tx used in unit tests.
type mockTx struct{}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) { return m, nil }
func (m *mockTx) Commit(ctx context.Context) error          { return nil }
func (m *mockTx) Rollback(ctx context.Context) error        { return nil }
func (m *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	panic("not implemented")
}
func (m *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("not implemented")
}
func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("not implemented")
}
func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	panic("not implemented")
}
func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	panic("not implemented")
}
func (m *mockTx) LargeObjects() pgx.LargeObjects { panic("not implemented") }
func (m *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	panic("not implemented")
}
func (m *mockTx) Conn() *pgx.Conn { panic("not implemented") }

// mockTxBeginner returns a mockTx for every Begin call.
type mockTxBeginner struct{}

func (m *mockTxBeginner) Begin(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil }

// --------------------------------------------------------------------------
// Mocks
// --------------------------------------------------------------------------

type mockPostRepo struct {
	create                func(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error)
	getByID               func(ctx context.Context, id uuid.UUID) (*entity.Post, error)
	getByAuthorWithCursor func(ctx context.Context, authorID uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, visibilityFilter string, limit int32) ([]*entity.Post, error)
	update                func(ctx context.Context, id uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error)
	delete                func(ctx context.Context, id uuid.UUID) error
}

func (m *mockPostRepo) WithTx(_ pgx.Tx) postRepo { return m }
func (m *mockPostRepo) Create(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error) {
	if m.create != nil {
		return m.create(ctx, authorID, content, visibility)
	}
	return &entity.Post{ID: uuid.New(), AuthorID: authorID, Content: content, Visibility: visibility, CreatedAt: time.Now()}, nil
}
func (m *mockPostRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Post, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, pkgerrors.ErrNotFound
}
func (m *mockPostRepo) GetByAuthorWithCursor(ctx context.Context, authorID uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, visibilityFilter string, limit int32) ([]*entity.Post, error) {
	if m.getByAuthorWithCursor != nil {
		return m.getByAuthorWithCursor(ctx, authorID, cursorCreatedAt, cursorPostID, visibilityFilter, limit)
	}
	return nil, nil
}
func (m *mockPostRepo) Update(ctx context.Context, id uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error) {
	if m.update != nil {
		return m.update(ctx, id, content, visibility)
	}
	return nil, pkgerrors.ErrNotFound
}
func (m *mockPostRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

type mockMediaRepo struct {
	add             func(ctx context.Context, postID uuid.UUID, key, mediaType string, position int32) (*entity.PostMedia, error)
	getByPost       func(ctx context.Context, postID uuid.UUID) ([]*entity.PostMedia, error)
	getByPostsBatch func(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]*entity.PostMedia, error)
}

func (m *mockMediaRepo) WithTx(_ pgx.Tx) mediaRepo { return m }
func (m *mockMediaRepo) Add(ctx context.Context, postID uuid.UUID, key, mediaType string, position int32) (*entity.PostMedia, error) {
	if m.add != nil {
		return m.add(ctx, postID, key, mediaType, position)
	}
	return &entity.PostMedia{ID: uuid.New(), PostID: postID, MediaKey: key, MediaType: mediaType}, nil
}
func (m *mockMediaRepo) GetByPost(ctx context.Context, postID uuid.UUID) ([]*entity.PostMedia, error) {
	if m.getByPost != nil {
		return m.getByPost(ctx, postID)
	}
	return nil, nil
}
func (m *mockMediaRepo) GetByPostsBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]*entity.PostMedia, error) {
	if m.getByPostsBatch != nil {
		return m.getByPostsBatch(ctx, postIDs)
	}
	return make(map[uuid.UUID][]*entity.PostMedia), nil
}

type mockLikeRepo struct {
	like            func(ctx context.Context, userID, postID uuid.UUID) error
	unlike          func(ctx context.Context, userID, postID uuid.UUID) error
	isLiked         func(ctx context.Context, userID, postID uuid.UUID) (bool, error)
	count           func(ctx context.Context, postID uuid.UUID) (int64, error)
	getLikedPostIDs func(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) ([]uuid.UUID, error)
}

func (m *mockLikeRepo) Like(ctx context.Context, userID, postID uuid.UUID) error {
	if m.like != nil {
		return m.like(ctx, userID, postID)
	}
	return nil
}
func (m *mockLikeRepo) Unlike(ctx context.Context, userID, postID uuid.UUID) error {
	if m.unlike != nil {
		return m.unlike(ctx, userID, postID)
	}
	return nil
}
func (m *mockLikeRepo) IsLiked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	if m.isLiked != nil {
		return m.isLiked(ctx, userID, postID)
	}
	return false, nil
}
func (m *mockLikeRepo) Count(ctx context.Context, postID uuid.UUID) (int64, error) {
	if m.count != nil {
		return m.count(ctx, postID)
	}
	return 0, nil
}
func (m *mockLikeRepo) GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) ([]uuid.UUID, error) {
	if m.getLikedPostIDs != nil {
		return m.getLikedPostIDs(ctx, userID, postIDs)
	}
	return nil, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func newPostService(pr postRepo, mr mediaRepo, lr likeRepo) *PostService {
	return &PostService{pool: &mockTxBeginner{}, postRepo: pr, mediaRepo: mr, likeRepo: lr}
}

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

func assertErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", code)
	}
	sentinels := map[string]error{
		"POST_NOT_FOUND":     post.ErrPostNotFound,
		"COMMENT_NOT_FOUND":  post.ErrCommentNotFound,
		"FORBIDDEN":          post.ErrForbidden,
		"SELF_LIKE":          post.ErrSelfLike,
		"EMPTY_CONTENT":      post.ErrEmptyContent,
		"INVALID_VISIBILITY": post.ErrInvalidVisibility,
	}
	if want, ok := sentinels[code]; ok {
		if err != want {
			t.Errorf("expected sentinel %q, got: %v", code, err)
		}
		return
	}
	t.Errorf("unknown sentinel code %q", code)
}

// --------------------------------------------------------------------------
// CreatePost tests
// --------------------------------------------------------------------------

func TestCreatePost_Success(t *testing.T) {
	authorID := uuid.New()
	pr := &mockPostRepo{}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	p, err := svc.CreatePost(context.Background(), authorID, "Hello world", entity.VisibilityPublic, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.AuthorID != authorID {
		t.Errorf("expected authorID %v, got %v", authorID, p.AuthorID)
	}
	if p.Content != "Hello world" {
		t.Errorf("expected content 'Hello world', got %q", p.Content)
	}
}

func TestCreatePost_EmptyContentAndNoMedia(t *testing.T) {
	svc := newPostService(&mockPostRepo{}, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.CreatePost(context.Background(), uuid.New(), "   ", entity.VisibilityPublic, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "EMPTY_CONTENT")
}

func TestCreatePost_WhitespaceOnlyContent_WithMedia_Succeeds(t *testing.T) {
	// Media present → allowed even with empty content
	svc := newPostService(&mockPostRepo{}, &mockMediaRepo{}, &mockLikeRepo{})

	p, err := svc.CreatePost(context.Background(), uuid.New(), "   ", entity.VisibilityPublic, []string{"media/img.jpg"}, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(p.Media) != 1 {
		t.Errorf("expected 1 media, got %d", len(p.Media))
	}
}

func TestCreatePost_InvalidVisibility(t *testing.T) {
	svc := newPostService(&mockPostRepo{}, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.CreatePost(context.Background(), uuid.New(), "content", "invalid", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "INVALID_VISIBILITY")
}

func TestCreatePost_ContentTrimmed(t *testing.T) {
	var savedContent string
	pr := &mockPostRepo{
		create: func(_ context.Context, _ uuid.UUID, content string, _ entity.Visibility) (*entity.Post, error) {
			savedContent = content
			return &entity.Post{ID: uuid.New(), Content: content, CreatedAt: time.Now()}, nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.CreatePost(context.Background(), uuid.New(), "  trimmed  ", entity.VisibilityPublic, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedContent != "trimmed" {
		t.Errorf("expected content to be trimmed, got %q", savedContent)
	}
}

func TestCreatePost_AttachesMedia(t *testing.T) {
	mediaAdded := 0
	mr := &mockMediaRepo{
		add: func(_ context.Context, _ uuid.UUID, key, _ string, _ int32) (*entity.PostMedia, error) {
			mediaAdded++
			return &entity.PostMedia{ID: uuid.New(), MediaKey: key}, nil
		},
	}
	svc := newPostService(&mockPostRepo{}, mr, &mockLikeRepo{})

	p, err := svc.CreatePost(context.Background(), uuid.New(), "post with media", entity.VisibilityPublic, []string{"img1.jpg", "img2.jpg"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mediaAdded != 2 {
		t.Errorf("expected 2 media adds, got %d", mediaAdded)
	}
	if len(p.Media) != 2 {
		t.Errorf("expected 2 media on post, got %d", len(p.Media))
	}
}

func TestCreatePost_InferMediaType_Video(t *testing.T) {
	var savedType string
	mr := &mockMediaRepo{
		add: func(_ context.Context, _ uuid.UUID, _, mediaType string, _ int32) (*entity.PostMedia, error) {
			savedType = mediaType
			return &entity.PostMedia{ID: uuid.New()}, nil
		},
	}
	svc := newPostService(&mockPostRepo{}, mr, &mockLikeRepo{})

	if _, err := svc.CreatePost(context.Background(), uuid.New(), "video post", entity.VisibilityPublic, []string{"clip.mp4"}, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedType != "video" {
		t.Errorf("expected media type 'video' for .mp4, got %q", savedType)
	}
}

func TestCreatePost_InferMediaType_Image(t *testing.T) {
	var savedType string
	mr := &mockMediaRepo{
		add: func(_ context.Context, _ uuid.UUID, _, mediaType string, _ int32) (*entity.PostMedia, error) {
			savedType = mediaType
			return &entity.PostMedia{ID: uuid.New()}, nil
		},
	}
	svc := newPostService(&mockPostRepo{}, mr, &mockLikeRepo{})

	if _, err := svc.CreatePost(context.Background(), uuid.New(), "image post", entity.VisibilityPublic, []string{"photo.jpg"}, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedType != "image" {
		t.Errorf("expected media type 'image' for .jpg, got %q", savedType)
	}
}

// --------------------------------------------------------------------------
// GetPost tests
// --------------------------------------------------------------------------

func TestGetPost_Success(t *testing.T) {
	authorID := uuid.New()
	postID := uuid.New()
	pr := &mockPostRepo{
		getByID: func(_ context.Context, id uuid.UUID) (*entity.Post, error) {
			return samplePost(authorID), nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	p, err := svc.GetPost(context.Background(), postID, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p == nil {
		t.Fatal("expected post, got nil")
	}
}

func TestGetPost_NotFound(t *testing.T) {
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return nil, pkgerrors.ErrNotFound
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.GetPost(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "POST_NOT_FOUND")
}

func TestGetPost_IsLiked_WhenViewerProvided(t *testing.T) {
	viewerID := uuid.New()
	postID := uuid.New()
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			p := samplePost(uuid.New())
			p.ID = postID
			p.LikeCount = 5
			return p, nil
		},
	}
	lr := &mockLikeRepo{
		getLikedPostIDs: func(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
			return ids, nil // viewer has liked all posts
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, lr)

	p, err := svc.GetPost(context.Background(), postID, &viewerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.IsLiked {
		t.Error("expected IsLiked=true when viewer has liked")
	}
	if p.LikeCount != 5 {
		t.Errorf("expected LikeCount=5, got %d", p.LikeCount)
	}
}

func TestGetPost_IsLiked_NilWhenNoViewer(t *testing.T) {
	getLikedCalled := false
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return samplePost(uuid.New()), nil
		},
	}
	lr := &mockLikeRepo{
		getLikedPostIDs: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]uuid.UUID, error) {
			getLikedCalled = true
			return nil, nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, lr)

	if _, err := svc.GetPost(context.Background(), uuid.New(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getLikedCalled {
		t.Error("GetLikedPostIDs should not be called when viewerID is nil")
	}
}

// --------------------------------------------------------------------------
// UpdatePost tests
// --------------------------------------------------------------------------

func TestUpdatePost_Success(t *testing.T) {
	authorID := uuid.New()
	postID := uuid.New()
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			p := samplePost(authorID)
			p.ID = postID
			return p, nil
		},
		update: func(_ context.Context, _ uuid.UUID, content string, v entity.Visibility) (*entity.Post, error) {
			return &entity.Post{ID: postID, AuthorID: authorID, Content: content, Visibility: v, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	p, err := svc.UpdatePost(context.Background(), postID, authorID, "Updated content", entity.VisibilityFollowers, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got %q", p.Content)
	}
}

func TestUpdatePost_NotFound(t *testing.T) {
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return nil, pkgerrors.ErrNotFound
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.UpdatePost(context.Background(), uuid.New(), uuid.New(), "content", entity.VisibilityPublic, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "POST_NOT_FOUND")
}

func TestUpdatePost_Forbidden_NotOwner(t *testing.T) {
	authorID := uuid.New()
	otherUserID := uuid.New()
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return samplePost(authorID), nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.UpdatePost(context.Background(), uuid.New(), otherUserID, "content", entity.VisibilityPublic, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "FORBIDDEN")
}

func TestUpdatePost_InvalidVisibility(t *testing.T) {
	authorID := uuid.New()
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return samplePost(authorID), nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	_, err := svc.UpdatePost(context.Background(), uuid.New(), authorID, "content", "bad", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "INVALID_VISIBILITY")
}

// --------------------------------------------------------------------------
// DeletePost tests
// --------------------------------------------------------------------------

func TestDeletePost_Success(t *testing.T) {
	authorID := uuid.New()
	deleteCalled := false
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return samplePost(authorID), nil
		},
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	err := svc.DeletePost(context.Background(), uuid.New(), authorID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Error("expected Delete to be called")
	}
}

func TestDeletePost_Forbidden_NotOwner(t *testing.T) {
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return samplePost(uuid.New()), nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	err := svc.DeletePost(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertErrorCode(t, err, "FORBIDDEN")
}

func TestDeletePost_NotFound(t *testing.T) {
	pr := &mockPostRepo{
		getByID: func(_ context.Context, _ uuid.UUID) (*entity.Post, error) {
			return nil, pkgerrors.ErrNotFound
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	err := svc.DeletePost(context.Background(), uuid.New(), uuid.New())
	assertErrorCode(t, err, "POST_NOT_FOUND")
}

// --------------------------------------------------------------------------
// GetUserPosts tests
// --------------------------------------------------------------------------

func TestGetUserPosts_Success(t *testing.T) {
	authorID := uuid.New()
	pr := &mockPostRepo{
		getByAuthorWithCursor: func(_ context.Context, _ uuid.UUID, _ pgtype.Timestamptz, _ uuid.UUID, _ string, _ int32) ([]*entity.Post, error) {
			return []*entity.Post{samplePost(authorID)}, nil
		},
	}
	svc := newPostService(pr, &mockMediaRepo{}, &mockLikeRepo{})

	posts, nextCursor, err := svc.GetUserPosts(context.Background(), authorID, nil, nil, "", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
	if nextCursor != nil {
		t.Errorf("expected no next cursor for single page, got %v", nextCursor)
	}
}

// --------------------------------------------------------------------------
// inferMediaType tests
// --------------------------------------------------------------------------

func TestInferMediaType(t *testing.T) {
	cases := []struct {
		key      string
		expected string
	}{
		{"video.mp4", "video"},
		{"VIDEO.MP4", "video"},
		{"clip.webm", "video"},
		{"photo.jpg", "image"},
		{"image.png", "image"},
		{"file.gif", "image"},
		{"no-extension", "image"},
	}
	for _, tc := range cases {
		got := inferMediaType(tc.key)
		if got != tc.expected {
			t.Errorf("inferMediaType(%q) = %q, want %q", tc.key, got, tc.expected)
		}
	}
}
