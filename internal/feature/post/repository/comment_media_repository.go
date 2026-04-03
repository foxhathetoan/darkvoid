package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/post/db"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

// CommentMediaRepository handles DB access for comment media attachments.
type CommentMediaRepository struct {
	queries *db.Queries
}

// NewCommentMediaRepository creates a new CommentMediaRepository.
func NewCommentMediaRepository(pool *pgxpool.Pool) *CommentMediaRepository {
	return &CommentMediaRepository{queries: db.New(pool)}
}

// WithTx returns a new CommentMediaRepository that executes queries within the given transaction.
func (r *CommentMediaRepository) WithTx(tx pgx.Tx) *CommentMediaRepository {
	return &CommentMediaRepository{queries: r.queries.WithTx(tx)}
}

// Add inserts a media attachment for a comment.
func (r *CommentMediaRepository) Add(ctx context.Context, commentID uuid.UUID, mediaKey, mediaType string, position int32) (*entity.CommentMedia, error) {
	row, err := r.queries.AddCommentMedia(ctx, db.AddCommentMediaParams{
		CommentID: commentID,
		MediaKey:  mediaKey,
		MediaType: mediaType,
		Position:  position,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToCommentMedia(row), nil
}

// GetByComment returns all media for a single comment, ordered by position.
func (r *CommentMediaRepository) GetByComment(ctx context.Context, commentID uuid.UUID) ([]*entity.CommentMedia, error) {
	rows, err := r.queries.GetCommentMedia(ctx, commentID)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make([]*entity.CommentMedia, len(rows))
	for i, row := range rows {
		result[i] = rowToCommentMedia(row)
	}
	return result, nil
}

// GetByCommentsBatch returns media grouped by comment ID for a batch of comments.
func (r *CommentMediaRepository) GetByCommentsBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]*entity.CommentMedia, error) {
	rows, err := r.queries.GetCommentMediaBatch(ctx, commentIDs)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make(map[uuid.UUID][]*entity.CommentMedia, len(commentIDs))
	for _, row := range rows {
		m := rowToCommentMedia(row)
		result[m.CommentID] = append(result[m.CommentID], m)
	}
	return result, nil
}

// DeleteAllByComment removes all media for a comment.
func (r *CommentMediaRepository) DeleteAllByComment(ctx context.Context, commentID uuid.UUID) error {
	return database.MapDBError(r.queries.DeleteAllCommentMedia(ctx, commentID))
}

func rowToCommentMedia(row db.PostCommentMedium) *entity.CommentMedia {
	return &entity.CommentMedia{
		ID:        row.ID,
		CommentID: row.CommentID,
		MediaKey:  row.MediaKey,
		MediaType: row.MediaType,
		Position:  row.Position,
		CreatedAt: row.CreatedAt.Time,
	}
}
