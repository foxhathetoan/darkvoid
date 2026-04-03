package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	postcache "github.com/jarviisha/darkvoid/internal/feature/post/cache"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/feature/post/repository"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

const (
	trendingHashtagsLimit = 20
	searchHashtagsLimit   = 10
	searchMinPrefixLen    = 2
)

// HashtagService handles hashtag business logic.
type HashtagService struct {
	hashtagRepo  hashtagRepo
	hashtagCache hashtagCache
	postRepo     postRepo
	userReader   userReader
}

// NewHashtagService creates a new HashtagService.
func NewHashtagService(
	hashtagRepo *repository.HashtagRepository,
	hashtagCache hashtagCache,
	postRepo *repository.PostRepository,
	userReader userReader,
) *HashtagService {
	return &HashtagService{
		hashtagRepo:  &hashtagRepoTxable{hashtagRepo},
		hashtagCache: hashtagCache,
		postRepo:     &postRepoTxable{postRepo},
		userReader:   userReader,
	}
}

// GetTrending returns trending hashtags, served from cache when available.
func (s *HashtagService) GetTrending(ctx context.Context) ([]*entity.TrendingHashtag, error) {
	if cached, err := s.hashtagCache.GetTrendingHashtags(ctx); err == nil && cached != nil {
		return cached, nil
	}

	tags, err := s.hashtagRepo.GetTrending(ctx, trendingHashtagsLimit)
	if err != nil {
		logger.LogError(ctx, err, "failed to get trending hashtags")
		return nil, errors.NewInternalError(err)
	}

	if err := s.hashtagCache.SetTrendingHashtags(ctx, tags); err != nil {
		logger.LogError(ctx, err, "failed to cache trending hashtags")
		// non-fatal: cache miss next time
	}

	return tags, nil
}

// SearchHashtags returns up to 10 hashtag names that start with prefix.
// prefix must be at least 2 characters. Results are cached for 2 minutes.
func (s *HashtagService) SearchHashtags(ctx context.Context, prefix string) ([]string, error) {
	if len(prefix) < searchMinPrefixLen {
		return nil, errors.NewBadRequestError("search prefix must be at least 2 characters")
	}

	if cached, err := s.hashtagCache.GetSearchResults(ctx, prefix); err == nil && cached != nil {
		return cached, nil
	}

	names, err := s.hashtagRepo.SearchByPrefix(ctx, prefix, searchHashtagsLimit)
	if err != nil {
		logger.LogError(ctx, err, "failed to search hashtags", "prefix", prefix)
		return nil, errors.NewInternalError(err)
	}

	if err := s.hashtagCache.SetSearchResults(ctx, prefix, names); err != nil {
		logger.LogError(ctx, err, "failed to cache hashtag search", "prefix", prefix)
	}

	return names, nil
}

// GetPostsByHashtag returns cursor-paginated public posts for a hashtag.
// Page 1 (cursor == nil) is served from cache when available (TTL 60s).
func (s *HashtagService) GetPostsByHashtag(ctx context.Context, name string, viewerID *uuid.UUID, cursor *post.UserPostCursor, limit int32) ([]*entity.Post, *post.UserPostCursor, error) {
	if limit <= 0 {
		limit = 20
	}

	// Serve page 1 from cache when available.
	if cursor == nil {
		if page, err := s.hashtagCache.GetHashtagPostsPage1(ctx, name); err == nil && page != nil {
			var nextCursor *post.UserPostCursor
			if page.NextCursor != "" {
				nextCursor, _ = post.DecodeUserPostCursor(page.NextCursor)
			}
			return page.Posts, nextCursor, nil
		}
	}

	cursorTS := pgtype.Timestamptz{Time: post.MaxUserPostTime, Valid: true}
	cursorID := uuid.Max
	if cursor != nil {
		cursorTS = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		var err error
		cursorID, err = uuid.Parse(cursor.PostID)
		if err != nil {
			return nil, nil, errors.NewBadRequestError("invalid cursor post_id")
		}
	}

	posts, err := s.hashtagRepo.GetPostsByHashtag(ctx, name, cursorTS, cursorID, limit+1)
	if err != nil {
		logger.LogError(ctx, err, "failed to get posts by hashtag", "hashtag", name)
		return nil, nil, errors.NewInternalError(err)
	}

	var nextCursor *post.UserPostCursor
	if len(posts) > int(limit) {
		last := posts[limit-1]
		nextCursor = &post.UserPostCursor{
			CreatedAt: last.CreatedAt,
			PostID:    last.ID.String(),
		}
		posts = posts[:limit]
	}

	s.enrichHashtagPosts(ctx, posts, viewerID)

	// Cache page 1 result (cursor was nil → this is the first page).
	if cursor == nil {
		encoded := ""
		if nextCursor != nil {
			encoded = nextCursor.Encode()
		}
		page := &postcache.HashtagPostsPage{Posts: posts, NextCursor: encoded}
		if err := s.hashtagCache.SetHashtagPostsPage1(ctx, name, page); err != nil {
			logger.LogError(ctx, err, "failed to cache hashtag posts page1", "hashtag", name)
		}
	}

	return posts, nextCursor, nil
}

// enrichHashtagPosts enriches a slice of posts with author info and tags.
func (s *HashtagService) enrichHashtagPosts(ctx context.Context, posts []*entity.Post, viewerID *uuid.UUID) {
	if len(posts) == 0 {
		return
	}

	// Batch-fetch tags
	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	if tagsMap, err := s.hashtagRepo.GetNamesByPostIDs(ctx, ids); err == nil {
		for _, p := range posts {
			if names, ok := tagsMap[p.ID]; ok {
				p.Tags = names
			}
		}
	}

	// Batch-fetch authors
	if s.userReader != nil {
		seen := make(map[uuid.UUID]bool, len(posts))
		authorIDs := make([]uuid.UUID, 0, len(posts))
		for _, p := range posts {
			if !seen[p.AuthorID] {
				seen[p.AuthorID] = true
				authorIDs = append(authorIDs, p.AuthorID)
			}
		}
		authors, err := s.userReader.GetAuthorsByIDs(ctx, authorIDs)
		if err != nil {
			logger.LogError(ctx, err, "failed to enrich hashtag post authors")
		} else {
			for _, p := range posts {
				if a, ok := authors[p.AuthorID]; ok {
					p.Author = a
				}
			}
		}
	}
	_ = viewerID // isLiked enrichment can be added when a likeRepo is injected
}
