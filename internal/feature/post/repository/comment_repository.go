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

type CommentRepository struct {
	queries *db.Queries
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{queries: db.New(pool)}
}

// WithTx returns a new CommentRepository that executes queries within the given transaction.
func (r *CommentRepository) WithTx(tx pgx.Tx) *CommentRepository {
	return &CommentRepository{queries: r.queries.WithTx(tx)}
}

func (r *CommentRepository) Create(ctx context.Context, postID, authorID uuid.UUID, parentID *uuid.UUID, content string) (*entity.Comment, error) {
	var pgParentID pgtype.UUID
	if parentID != nil {
		pgParentID = pgtype.UUID{Bytes: *parentID, Valid: true}
	}
	row, err := r.queries.CreateComment(ctx, db.CreateCommentParams{
		PostID:   postID,
		AuthorID: authorID,
		ParentID: pgParentID,
		Content:  content,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToComment(row), nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Comment, error) {
	row, err := r.queries.GetCommentByID(ctx, id)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToComment(row), nil
}

func (r *CommentRepository) GetByPost(ctx context.Context, postID uuid.UUID, limit, offset int32) ([]*entity.Comment, error) {
	rows, err := r.queries.GetCommentsByPost(ctx, db.GetCommentsByPostParams{
		PostID: postID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToComments(rows), nil
}

func (r *CommentRepository) GetReplies(ctx context.Context, parentID uuid.UUID, limit, offset int32) ([]*entity.Comment, error) {
	rows, err := r.queries.GetReplies(ctx, db.GetRepliesParams{
		ParentID: pgtype.UUID{Bytes: parentID, Valid: true},
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToComments(rows), nil
}

func (r *CommentRepository) GetReplyCountsBatch(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	rows, err := r.queries.GetReplyCountsBatch(ctx, parentIDs)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		result[uuid.UUID(row.RootID.Bytes)] = row.Count
	}
	return result, nil
}

func (r *CommentRepository) GetRepliesPreview(ctx context.Context, parentIDs []uuid.UUID, limitPerParent int32) (map[uuid.UUID][]*entity.Comment, error) {
	rows, err := r.queries.GetRepliesPreview(ctx, db.GetRepliesPreviewParams{
		Column1: parentIDs,
		Column2: limitPerParent,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make(map[uuid.UUID][]*entity.Comment, len(parentIDs))
	for _, row := range rows {
		if !row.ParentID.Valid {
			continue
		}
		c := rowToComment(row)
		pid := uuid.UUID(row.ParentID.Bytes)
		result[pid] = append(result[pid], c)
	}
	return result, nil
}

func (r *CommentRepository) CountByPost(ctx context.Context, postID uuid.UUID) (int64, error) {
	count, err := r.queries.CountCommentsByPost(ctx, postID)
	if err != nil {
		return 0, database.MapDBError(err)
	}
	return count, nil
}

func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return database.MapDBError(r.queries.DeleteComment(ctx, id))
}

func rowToComment(row db.PostComment) *entity.Comment {
	c := &entity.Comment{
		ID:        row.ID,
		PostID:    row.PostID,
		AuthorID:  row.AuthorID,
		Content:   row.Content,
		LikeCount: row.LikeCount,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
	if row.ParentID.Valid {
		id := uuid.UUID(row.ParentID.Bytes)
		c.ParentID = &id
	}
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		c.DeletedAt = &t
	}
	return c
}

func rowsToComments(rows []db.PostComment) []*entity.Comment {
	result := make([]*entity.Comment, len(rows))
	for i, row := range rows {
		result[i] = rowToComment(row)
	}
	return result
}
