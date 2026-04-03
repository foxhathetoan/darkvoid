package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/post/db"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

type LikeRepository struct {
	queries *db.Queries
}

func NewLikeRepository(pool *pgxpool.Pool) *LikeRepository {
	return &LikeRepository{queries: db.New(pool)}
}

func (r *LikeRepository) Like(ctx context.Context, userID, postID uuid.UUID) error {
	return database.MapDBError(r.queries.LikePost(ctx, db.LikePostParams{
		UserID: userID,
		PostID: postID,
	}))
}

func (r *LikeRepository) Unlike(ctx context.Context, userID, postID uuid.UUID) error {
	return database.MapDBError(r.queries.UnlikePost(ctx, db.UnlikePostParams{
		UserID: userID,
		PostID: postID,
	}))
}

func (r *LikeRepository) IsLiked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	ok, err := r.queries.IsLiked(ctx, db.IsLikedParams{
		UserID: userID,
		PostID: postID,
	})
	if err != nil {
		return false, database.MapDBError(err)
	}
	return ok, nil
}

func (r *LikeRepository) Count(ctx context.Context, postID uuid.UUID) (int64, error) {
	count, err := r.queries.CountLikes(ctx, postID)
	if err != nil {
		return 0, database.MapDBError(err)
	}
	return count, nil
}

func (r *LikeRepository) GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.queries.GetLikedPostIDs(ctx, db.GetLikedPostIDsParams{
		UserID:  userID,
		Column2: postIDs,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return ids, nil
}
