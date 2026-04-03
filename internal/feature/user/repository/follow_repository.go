package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

type FollowRepository struct {
	queries *db.Queries
}

func NewFollowRepository(pool *pgxpool.Pool) *FollowRepository {
	return &FollowRepository{queries: db.New(pool)}
}

func (r *FollowRepository) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return database.MapDBError(r.queries.Follow(ctx, db.FollowParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}))
}

func (r *FollowRepository) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return database.MapDBError(r.queries.Unfollow(ctx, db.UnfollowParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}))
}

func (r *FollowRepository) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	ok, err := r.queries.IsFollowing(ctx, db.IsFollowingParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
	if err != nil {
		return false, database.MapDBError(err)
	}
	return ok, nil
}

func (r *FollowRepository) GetFollowers(ctx context.Context, followeeID uuid.UUID, limit, offset int32) ([]*entity.Follow, error) {
	rows, err := r.queries.GetFollowers(ctx, db.GetFollowersParams{
		FolloweeID: followeeID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToFollows(rows), nil
}

func (r *FollowRepository) GetFollowing(ctx context.Context, followerID uuid.UUID, limit, offset int32) ([]*entity.Follow, error) {
	rows, err := r.queries.GetFollowing(ctx, db.GetFollowingParams{
		FollowerID: followerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowsToFollows(rows), nil
}

func (r *FollowRepository) CountFollowers(ctx context.Context, followeeID uuid.UUID) (int64, error) {
	count, err := r.queries.CountFollowers(ctx, followeeID)
	if err != nil {
		return 0, database.MapDBError(err)
	}
	return count, nil
}

func (r *FollowRepository) CountFollowing(ctx context.Context, followerID uuid.UUID) (int64, error) {
	count, err := r.queries.CountFollowing(ctx, followerID)
	if err != nil {
		return 0, database.MapDBError(err)
	}
	return count, nil
}

func rowsToFollows(rows []db.UsrFollow) []*entity.Follow {
	result := make([]*entity.Follow, len(rows))
	for i, row := range rows {
		result[i] = &entity.Follow{
			FollowerID: row.FollowerID,
			FolloweeID: row.FolloweeID,
			CreatedAt:  row.CreatedAt.Time,
		}
	}
	return result
}
