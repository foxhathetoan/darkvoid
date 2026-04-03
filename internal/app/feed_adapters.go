package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jarviisha/darkvoid/internal/feature/feed"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
	postentity "github.com/jarviisha/darkvoid/internal/feature/post/entity"
	postrepo "github.com/jarviisha/darkvoid/internal/feature/post/repository"
	userrepo "github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	pkgerrors "github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// --- Entity conversion ---

// toFeedPost converts a post.entity.Post to feedentity.Post.
// This conversion lives exclusively at the app layer — the only place allowed to know both contexts.
func toFeedPost(p *postentity.Post) *feedentity.Post {
	media := make([]feedentity.PostMedia, len(p.Media))
	for i, m := range p.Media {
		media[i] = feedentity.PostMedia{
			ID:        m.ID,
			PostID:    m.PostID,
			MediaKey:  m.MediaKey,
			MediaType: m.MediaType,
			Position:  m.Position,
			CreatedAt: m.CreatedAt,
		}
	}
	return &feedentity.Post{
		ID:           p.ID,
		AuthorID:     p.AuthorID,
		Content:      p.Content,
		Visibility:   string(p.Visibility),
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		Media:        media,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		IsLiked:      p.IsLiked,
	}
}

// --- userReader ---

// userReader implements feed.UserReader using UserRepository.
type userReader struct {
	userRepo *userrepo.UserRepository
}

func (r *userReader) GetAuthorsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*feedentity.Author, error) {
	users, err := r.userRepo.GetUsersByIDsAny(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]*feedentity.Author, len(users))
	for _, u := range users {
		result[u.ID] = &feedentity.Author{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarKey:   u.AvatarKey,
		}
	}
	return result, nil
}

// --- postReader ---

// postReader implements feed.PostReader using post repositories directly.
type postReader struct {
	postRepo   *postrepo.PostRepository
	mediaRepo  *postrepo.MediaRepository
	likeRepo   *postrepo.LikeRepository
	userReader feed.UserReader
}

func (r *postReader) GetFollowingPostsWithCursor(ctx context.Context, authorIDs []uuid.UUID, cursor *feed.FollowingCursor, limit int32) ([]*feedentity.Post, error) {
	var cursorTS pgtype.Timestamptz
	var cursorID uuid.UUID

	if cursor != nil {
		var err error
		cursorTS, cursorID, err = cursor.PgParams()
		if err != nil {
			return nil, pkgerrors.NewBadRequestError("invalid cursor post_id")
		}
	} else {
		cursorTS, cursorID = feed.DefaultDiscoverPgParams()
	}

	posts, err := r.postRepo.GetFollowingPostsWithCursor(ctx, authorIDs, cursorTS, cursorID, limit)
	if err != nil {
		return nil, pkgerrors.NewInternalError(err)
	}
	return r.convertAndEnrich(ctx, posts), nil
}

func (r *postReader) GetTrendingPosts(ctx context.Context, limit int32) ([]*feedentity.Post, error) {
	posts, err := r.postRepo.GetTrendingPosts(ctx, limit)
	if err != nil {
		return nil, pkgerrors.NewInternalError(err)
	}
	return r.convertAndEnrich(ctx, posts), nil
}

func (r *postReader) GetDiscoverWithCursor(ctx context.Context, cursor *feed.DiscoverCursor, limit int32, viewerID *uuid.UUID) ([]*feedentity.Post, error) {
	var cursorTS pgtype.Timestamptz
	var cursorID uuid.UUID

	if cursor != nil {
		var err error
		cursorTS, cursorID, err = cursor.PgParams()
		if err != nil {
			return nil, pkgerrors.NewBadRequestError("invalid cursor post_id")
		}
	} else {
		cursorTS, cursorID = feed.DefaultDiscoverPgParams()
	}

	posts, err := r.postRepo.GetDiscoverWithCursor(ctx, cursorTS, cursorID, limit)
	if err != nil {
		logger.LogError(ctx, err, "failed to get discover feed")
		return nil, pkgerrors.NewInternalError(err)
	}
	result := r.convertAndEnrich(ctx, posts)
	if viewerID != nil {
		r.enrichIsLiked(ctx, *viewerID, result)
	}
	return result, nil
}

// convertAndEnrich converts post entities and enriches them with media and author info in batch.
// Is-liked enrichment is handled by the caller when a viewerID is available.
func (r *postReader) convertAndEnrich(ctx context.Context, posts []*postentity.Post) []*feedentity.Post {
	if len(posts) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	mediaMap, err := r.mediaRepo.GetByPostsBatch(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to batch fetch post media for feed")
	} else {
		for _, p := range posts {
			if media, ok := mediaMap[p.ID]; ok {
				p.Media = media
			}
		}
	}

	result := make([]*feedentity.Post, len(posts))
	for i, p := range posts {
		result[i] = toFeedPost(p)
	}
	r.enrichAuthors(ctx, result)
	return result
}

// enrichIsLiked batch-fetches like status for the viewer and sets Post.IsLiked.
// Best-effort: on error, posts are returned as-is.
func (r *postReader) enrichIsLiked(ctx context.Context, viewerID uuid.UUID, posts []*feedentity.Post) {
	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	likedIDs, err := r.likeRepo.GetLikedPostIDs(ctx, viewerID, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to batch fetch liked post IDs for feed")
		return
	}
	likedSet := make(map[uuid.UUID]bool, len(likedIDs))
	for _, id := range likedIDs {
		likedSet[id] = true
	}
	for _, p := range posts {
		p.IsLiked = likedSet[p.ID]
	}
}

// enrichAuthors batch-fetches author info for the given posts and sets Post.Author.
// Best-effort: on error, posts are returned without author info rather than failing the request.
func (r *postReader) enrichAuthors(ctx context.Context, posts []*feedentity.Post) {
	if len(posts) == 0 {
		return
	}

	// Collect unique author IDs
	seen := make(map[uuid.UUID]bool, len(posts))
	ids := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		if !seen[p.AuthorID] {
			seen[p.AuthorID] = true
			ids = append(ids, p.AuthorID)
		}
	}

	authors, err := r.userReader.GetAuthorsByIDs(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich post authors")
		return
	}

	for _, p := range posts {
		if a, ok := authors[p.AuthorID]; ok {
			p.Author = a
		}
	}
}

// --- followReader ---

// followReader implements feed.FollowReader using FollowService.
type followReader struct {
	followService *service.FollowService
}

func (r *followReader) GetFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return r.followService.GetFollowingIDs(ctx, userID)
}

// --- likeReader ---

// likeReader implements feed.LikeReader using LikeRepository.
type likeReader struct {
	likeRepo *postrepo.LikeRepository
}

func (r *likeReader) GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) ([]uuid.UUID, error) {
	return r.likeRepo.GetLikedPostIDs(ctx, userID, postIDs)
}
