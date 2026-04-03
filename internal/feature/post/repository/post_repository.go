package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/post/db"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

type PostRepository struct {
	queries *db.Queries
}

func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{queries: db.New(pool)}
}

// WithTx returns a new PostRepository that executes queries within the given transaction.
func (r *PostRepository) WithTx(tx pgx.Tx) *PostRepository {
	return &PostRepository{queries: r.queries.WithTx(tx)}
}

func (r *PostRepository) Create(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error) {
	row, err := r.queries.CreatePost(ctx, db.CreatePostParams{
		AuthorID:   authorID,
		Content:    content,
		Visibility: string(visibility),
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToPost(row), nil
}

func (r *PostRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Post, error) {
	row, err := r.queries.GetPostByID(ctx, id)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToPost(row), nil
}

func (r *PostRepository) Update(ctx context.Context, id uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error) {
	row, err := r.queries.UpdatePost(ctx, db.UpdatePostParams{
		ID:         id,
		Content:    content,
		Visibility: string(visibility),
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToPost(row), nil
}

func (r *PostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return database.MapDBError(r.queries.DeletePost(ctx, id))
}

func (r *PostRepository) GetFollowingPostsWithCursor(ctx context.Context, authorIDs []uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, limit int32) ([]*entity.Post, error) {
	rows, err := r.queries.GetFollowingPostsWithCursor(ctx, db.GetFollowingPostsWithCursorParams{
		Column1: authorIDs,
		Column2: cursorCreatedAt,
		Column3: cursorPostID,
		Limit:   limit,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return followingCursorRowsToPosts(rows), nil
}

func (r *PostRepository) GetTrendingPosts(ctx context.Context, limit int32) ([]*entity.Post, error) {
	rows, err := r.queries.GetTrendingPosts(ctx, limit)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToPosts(rows), nil
}

func (r *PostRepository) GetByAuthorWithCursor(ctx context.Context, authorID uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, visibilityFilter string, limit int32) ([]*entity.Post, error) {
	rows, err := r.queries.GetUserPostsWithCursor(ctx, db.GetUserPostsWithCursorParams{
		AuthorID: authorID,
		Column2:  cursorCreatedAt,
		Column3:  cursorPostID,
		Column4:  visibilityFilter,
		Limit:    limit,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToPosts(rows), nil
}

func (r *PostRepository) GetDiscoverWithCursor(ctx context.Context, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, limit int32) ([]*entity.Post, error) {
	rows, err := r.queries.GetDiscoverWithCursor(ctx, db.GetDiscoverWithCursorParams{
		CursorCreatedAt: cursorCreatedAt,
		CursorPostID:    cursorPostID,
		Limit:           limit,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToPosts(rows), nil
}

func rowToPost(row db.PostPost) *entity.Post {
	p := &entity.Post{
		ID:           row.ID,
		AuthorID:     row.AuthorID,
		Content:      row.Content,
		Visibility:   entity.Visibility(row.Visibility),
		LikeCount:    row.LikeCount,
		CommentCount: row.CommentCount,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		p.DeletedAt = &t
	}
	return p
}

func rowsToPosts(rows []db.PostPost) []*entity.Post {
	result := make([]*entity.Post, len(rows))
	for i, row := range rows {
		result[i] = rowToPost(row)
	}
	return result
}

func followingCursorRowsToPosts(rows []db.GetFollowingPostsWithCursorRow) []*entity.Post {
	result := make([]*entity.Post, len(rows))
	for i, row := range rows {
		p := &entity.Post{
			ID:           row.ID,
			AuthorID:     row.AuthorID,
			Content:      row.Content,
			Visibility:   entity.Visibility(row.Visibility),
			LikeCount:    row.LikeCount,
			CommentCount: row.CommentCount,
			CreatedAt:    row.CreatedAt.Time,
			UpdatedAt:    row.UpdatedAt.Time,
		}
		if row.DeletedAt.Valid {
			t := row.DeletedAt.Time
			p.DeletedAt = &t
		}
		result[i] = p
	}
	return result
}

func hashtagCursorRowsToPosts(rows []db.GetPostsByHashtagWithCursorRow) []*entity.Post {
	result := make([]*entity.Post, len(rows))
	for i, row := range rows {
		p := &entity.Post{
			ID:           row.ID,
			AuthorID:     row.AuthorID,
			Content:      row.Content,
			Visibility:   entity.Visibility(row.Visibility),
			LikeCount:    row.LikeCount,
			CommentCount: row.CommentCount,
			CreatedAt:    row.CreatedAt.Time,
			UpdatedAt:    row.UpdatedAt.Time,
		}
		if row.DeletedAt.Valid {
			t := row.DeletedAt.Time
			p.DeletedAt = &t
		}
		result[i] = p
	}
	return result
}
