package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/jarviisha/darkvoid/internal/feature/search/dto"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// userSearcher fetches users matching a query string.
// Implemented at the app layer to avoid cross-context imports.
type userSearcher interface {
	SearchByQuery(ctx context.Context, query string, limit, offset int32) ([]dto.UserResult, error)
}

// postSearcher fetches posts matching a query string via full-text search.
// Implemented at the app layer to avoid cross-context imports.
type postSearcher interface {
	SearchByQuery(ctx context.Context, query string, limit, offset int32) ([]dto.PostResult, error)
}

// hashtagSearcher fetches hashtag names matching a prefix.
// Implemented at the app layer to avoid cross-context imports.
type hashtagSearcher interface {
	SearchByPrefix(ctx context.Context, prefix string, limit int32) ([]string, error)
}

// SearchService aggregates search results across users, posts, and hashtags.
type SearchService struct {
	users    userSearcher
	posts    postSearcher
	hashtags hashtagSearcher
}

// NewSearchService creates a new SearchService.
func NewSearchService(users userSearcher, posts postSearcher, hashtags hashtagSearcher) *SearchService {
	return &SearchService{users: users, posts: posts, hashtags: hashtags}
}

const (
	defaultLimit = int32(20)
	maxLimit     = int32(50)
	// When type=all, each category gets a smaller slice to keep response lean.
	allModeLimit = int32(5)
)

// Search runs a unified search across one or all entity types.
func (s *SearchService) Search(ctx context.Context, query string, searchType dto.SearchType, limit, offset int32) (*dto.SearchResponse, error) {
	if query == "" {
		return nil, errors.NewBadRequestError("search query is required")
	}
	if len(query) < 2 {
		return nil, errors.NewBadRequestError("search query must be at least 2 characters")
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	resp := &dto.SearchResponse{Query: query, Type: string(searchType)}

	switch searchType {
	case dto.SearchTypeUsers:
		users, err := s.users.SearchByQuery(ctx, query, limit, offset)
		if err != nil {
			logger.LogError(ctx, err, "user search failed", "query", query)
			return nil, errors.NewInternalError(err)
		}
		resp.Users = users

	case dto.SearchTypePosts:
		posts, err := s.posts.SearchByQuery(ctx, query, limit, offset)
		if err != nil {
			logger.LogError(ctx, err, "post search failed", "query", query)
			return nil, errors.NewInternalError(err)
		}
		resp.Posts = posts

	case dto.SearchTypeHashtags:
		tags, err := s.hashtags.SearchByPrefix(ctx, query, limit)
		if err != nil {
			logger.LogError(ctx, err, "hashtag search failed", "query", query)
			return nil, errors.NewInternalError(err)
		}
		resp.Hashtags = tags

	default:
		// type=all — run all three in parallel, each limited to allModeLimit
		if err := s.searchAll(ctx, query, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// searchAll fetches users, posts, and hashtags in parallel.
func (s *SearchService) searchAll(ctx context.Context, query string, resp *dto.SearchResponse) error {
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		users, err := s.users.SearchByQuery(gctx, query, allModeLimit, 0)
		if err != nil {
			logger.LogError(ctx, err, "user search failed in all-mode", "query", query)
			return nil // non-fatal: degrade gracefully
		}
		resp.Users = users
		return nil
	})

	g.Go(func() error {
		posts, err := s.posts.SearchByQuery(gctx, query, allModeLimit, 0)
		if err != nil {
			logger.LogError(ctx, err, "post search failed in all-mode", "query", query)
			return nil
		}
		resp.Posts = posts
		return nil
	})

	g.Go(func() error {
		tags, err := s.hashtags.SearchByPrefix(gctx, query, allModeLimit)
		if err != nil {
			logger.LogError(ctx, err, "hashtag search failed in all-mode", "query", query)
			return nil
		}
		resp.Hashtags = tags
		return nil
	})

	return g.Wait()
}
