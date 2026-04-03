package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

type UserRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *UserRepository) ExistsUsername(ctx context.Context, username string) (bool, error) {
	exists, err := r.queries.ExistsUsername(ctx, username)
	if err != nil {
		return false, database.MapDBError(err)
	}
	return exists, nil
}

func (r *UserRepository) ExistsEmail(ctx context.Context, email string) (bool, error) {
	exists, err := r.queries.ExistsEmail(ctx, email)
	if err != nil {
		return false, database.MapDBError(err)
	}
	return exists, nil
}

func (r *UserRepository) ExistsEmailExcludingUser(ctx context.Context, email string, userID uuid.UUID) (bool, error) {
	exists, err := r.queries.ExistsEmailExcludingUser(ctx, db.ExistsEmailExcludingUserParams{
		Email: email,
		ID:    userID,
	})
	if err != nil {
		return false, database.MapDBError(err)
	}
	return exists, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	var createdBy pgtype.UUID
	if user.CreatedBy != nil {
		createdBy = pgtype.UUID{Bytes: *user.CreatedBy, Valid: true}
	}

	dbUser, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		CreatedBy:    createdBy,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}

	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	dbUser, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	dbUser, err := r.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, id uuid.UUID, email *string, updatedBy *uuid.UUID) (*entity.User, error) {
	var updatedByParam pgtype.UUID
	if updatedBy != nil {
		updatedByParam = pgtype.UUID{Bytes: *updatedBy, Valid: true}
	}

	dbUser, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:        id,
		Email:     email,
		UpdatedBy: updatedByParam,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) UpdateUserProfile(ctx context.Context, id uuid.UUID, params db.UpdateUserProfileParams) (*entity.User, error) {
	params.ID = id
	dbUser, err := r.queries.UpdateUserProfile(ctx, params)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string, updatedBy *uuid.UUID) error {
	var updatedByParam pgtype.UUID
	if updatedBy != nil {
		updatedByParam = pgtype.UUID{Bytes: *updatedBy, Valid: true}
	}

	return r.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
		UpdatedBy:    updatedByParam,
	})
}

func (r *UserRepository) DeactivateUser(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error {
	var updatedByParam pgtype.UUID
	if updatedBy != nil {
		updatedByParam = pgtype.UUID{Bytes: *updatedBy, Valid: true}
	}

	return r.queries.DeactivateUser(ctx, db.DeactivateUserParams{
		ID:        id,
		UpdatedBy: updatedByParam,
	})
}

// GetUserByIDAny fetches a user by ID regardless of active status.
// Use for admin lookups or any context where inactive users must still be resolved.
func (r *UserRepository) GetUserByIDAny(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	dbUser, err := r.queries.GetUserByIDAny(ctx, id)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToEntity(dbUser), nil
}

func (r *UserRepository) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	dbUsers, err := r.queries.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, database.MapDBError(err)
	}

	users := make([]*entity.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = dbUserToEntity(dbUser)
	}
	return users, nil
}

// GetUsersByIDsAny batch-fetches users by ID regardless of active status.
// Use for post author enrichment where authors may have been deactivated after posting.
func (r *UserRepository) GetUsersByIDsAny(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	dbUsers, err := r.queries.GetUsersByIDsAny(ctx, ids)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	users := make([]*entity.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = dbUserToEntity(dbUser)
	}
	return users, nil
}

// GetUsersByUsernames batch-fetches active users by their usernames.
func (r *UserRepository) GetUsersByUsernames(ctx context.Context, usernames []string) ([]*entity.User, error) {
	dbUsers, err := r.queries.GetUsersByUsernames(ctx, usernames)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	users := make([]*entity.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = dbUserToEntity(dbUser)
	}
	return users, nil
}

// SearchByQuery searches active users by username or display_name, ordered by relevance.
func (r *UserRepository) SearchByQuery(ctx context.Context, query string, limit, offset int32) ([]*entity.User, error) {
	dbUsers, err := r.queries.SearchUsersByQuery(ctx, db.SearchUsersByQueryParams{
		Query:  query,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	users := make([]*entity.User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = dbUserToEntity(u)
	}
	return users, nil
}

func (r *UserRepository) WithTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := r.queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func dbUserToEntity(dbUser db.UsrUser) *entity.User {
	u := &entity.User{
		ID:             dbUser.ID,
		Username:       dbUser.Username,
		Email:          dbUser.Email,
		PasswordHash:   dbUser.PasswordHash,
		IsActive:       dbUser.IsActive,
		DisplayName:    dbUser.DisplayName,
		Bio:            dbUser.Bio,
		AvatarKey:      dbUser.AvatarKey,
		CoverKey:       dbUser.CoverKey,
		Website:        dbUser.Website,
		Location:       dbUser.Location,
		CreatedAt:      dbUser.CreatedAt.Time,
		FollowerCount:  dbUser.FollowerCount,
		FollowingCount: dbUser.FollowingCount,
	}

	if dbUser.UpdatedAt.Valid {
		t := dbUser.UpdatedAt.Time
		u.UpdatedAt = &t
	}
	if dbUser.CreatedBy.Valid {
		id := uuid.UUID(dbUser.CreatedBy.Bytes)
		u.CreatedBy = &id
	}
	if dbUser.UpdatedBy.Valid {
		id := uuid.UUID(dbUser.UpdatedBy.Bytes)
		u.UpdatedBy = &id
	}

	return u
}
