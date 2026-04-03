package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/post/db"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

// MentionRepository handles persistence of post mentions.
type MentionRepository struct {
	queries *db.Queries
}

// NewMentionRepository creates a new MentionRepository.
func NewMentionRepository(pool *pgxpool.Pool) *MentionRepository {
	return &MentionRepository{queries: db.New(pool)}
}

// Insert adds a mention row, ignoring duplicates.
func (r *MentionRepository) Insert(ctx context.Context, postID, userID uuid.UUID) error {
	return database.MapDBError(r.queries.InsertMention(ctx, db.InsertMentionParams{
		PostID: postID,
		UserID: userID,
	}))
}

// DeleteByPost removes all mentions for a post.
func (r *MentionRepository) DeleteByPost(ctx context.Context, postID uuid.UUID) error {
	return database.MapDBError(r.queries.DeleteMentionsByPost(ctx, postID))
}

// GetBatch returns mention rows grouped by post ID for a set of posts.
func (r *MentionRepository) GetBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	rows, err := r.queries.GetMentionsBatch(ctx, postIDs)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make(map[uuid.UUID][]uuid.UUID, len(postIDs))
	for _, row := range rows {
		result[row.PostID] = append(result[row.PostID], row.UserID)
	}
	return result, nil
}

// GetByPost returns the user IDs of all users mentioned in a post.
func (r *MentionRepository) GetByPost(ctx context.Context, postID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.queries.GetMentionsByPost(ctx, postID)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.UserID
	}
	return ids, nil
}
