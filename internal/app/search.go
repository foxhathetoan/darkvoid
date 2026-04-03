package app

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/app/middleware"
	postentity "github.com/jarviisha/darkvoid/internal/feature/post/entity"
	postrepo "github.com/jarviisha/darkvoid/internal/feature/post/repository"
	searchdto "github.com/jarviisha/darkvoid/internal/feature/search/dto"
	searchhandler "github.com/jarviisha/darkvoid/internal/feature/search/handler"
	searchsvc "github.com/jarviisha/darkvoid/internal/feature/search/service"
	userentity "github.com/jarviisha/darkvoid/internal/feature/user/entity"
	userrepo "github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// SearchContext holds all search-related dependencies.
type SearchContext struct {
	handler *searchhandler.SearchHandler
}

// SetupSearchContext wires the unified search bounded context.
func SetupSearchContext(
	pool *pgxpool.Pool,
	userRepo *userrepo.UserRepository,
	hashtagRepo *postrepo.HashtagRepository,
	store storage.Storage,
) *SearchContext {
	postSearchRepo := postrepo.NewPostSearchRepository(pool)
	svc := searchsvc.NewSearchService(
		&searchUserAdapter{repo: userRepo, store: store},
		&searchPostAdapter{repo: postSearchRepo},
		&searchHashtagAdapter{repo: hashtagRepo},
	)
	return &SearchContext{handler: searchhandler.NewSearchHandler(svc)}
}

// RegisterRoutes mounts search routes. Search is public (no auth required).
func (c *SearchContext) RegisterRoutes(r chi.Router, auth middleware.AuthMiddleware) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimitByIP(120, time.Minute))
		r.Get("/search", c.handler.Search)
	})
}

// --------------------------------------------------------------------------
// Adapters — translate feature-specific types to search dto types
// --------------------------------------------------------------------------

// searchUserAdapter implements searchsvc.userSearcher using UserRepository.
type searchUserAdapter struct {
	repo  *userrepo.UserRepository
	store storage.Storage
}

func (a *searchUserAdapter) SearchByQuery(ctx context.Context, query string, limit, offset int32) ([]searchdto.UserResult, error) {
	users, err := a.repo.SearchByQuery(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return usersToSearchResults(users, a.store), nil
}

// searchPostAdapter implements searchsvc.postSearcher using PostSearchRepository.
type searchPostAdapter struct {
	repo *postrepo.PostSearchRepository
}

func (a *searchPostAdapter) SearchByQuery(ctx context.Context, query string, limit, offset int32) ([]searchdto.PostResult, error) {
	posts, err := a.repo.SearchByQuery(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return postsToSearchResults(posts), nil
}

// searchHashtagAdapter implements searchsvc.hashtagSearcher using HashtagRepository.
type searchHashtagAdapter struct {
	repo *postrepo.HashtagRepository
}

func (a *searchHashtagAdapter) SearchByPrefix(ctx context.Context, prefix string, limit int32) ([]string, error) {
	return a.repo.SearchByPrefix(ctx, prefix, limit)
}

// --------------------------------------------------------------------------
// Conversion helpers
// --------------------------------------------------------------------------

func usersToSearchResults(users []*userentity.User, store storage.Storage) []searchdto.UserResult {
	results := make([]searchdto.UserResult, 0, len(users))
	for _, u := range users {
		r := searchdto.UserResult{
			ID:            u.ID.String(),
			Username:      u.Username,
			DisplayName:   u.DisplayName,
			FollowerCount: u.FollowerCount,
		}
		if u.AvatarKey != nil {
			url := store.URL(*u.AvatarKey)
			r.AvatarURL = &url
		}
		results = append(results, r)
	}
	return results
}

func postsToSearchResults(posts []*postentity.Post) []searchdto.PostResult {
	results := make([]searchdto.PostResult, 0, len(posts))
	for _, p := range posts {
		results = append(results, searchdto.PostResult{
			ID:           p.ID.String(),
			AuthorID:     p.AuthorID.String(),
			Content:      p.Content,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CreatedAt:    p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return results
}
