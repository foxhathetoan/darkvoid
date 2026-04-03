package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/post/db"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

// PostSearchRepository queries posts by full-text search.
type PostSearchRepository struct {
	queries *db.Queries
}

// NewPostSearchRepository creates a new PostSearchRepository.
func NewPostSearchRepository(pool *pgxpool.Pool) *PostSearchRepository {
	return &PostSearchRepository{queries: db.New(pool)}
}

// SearchByQuery returns public posts whose content matches query, ordered by relevance.
func (r *PostSearchRepository) SearchByQuery(ctx context.Context, query string, limit, offset int32) ([]*entity.Post, error) {
	rows, err := r.queries.SearchPosts(ctx, db.SearchPostsParams{
		Query:  query,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	posts := make([]*entity.Post, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, rowToPost(row))
	}
	return posts, nil
}
