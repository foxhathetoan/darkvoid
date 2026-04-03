package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
)

type RefreshTokenRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) (*entity.RefreshToken, error) {
	dbToken, err := r.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return dbRefreshTokenToEntity(dbToken), nil
}

func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	dbToken, err := r.queries.GetRefreshTokenByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return dbRefreshTokenToEntity(dbToken), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	return r.queries.RevokeRefreshToken(ctx, token)
}

func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return r.queries.RevokeAllUserRefreshTokens(ctx, userID)
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	return r.queries.DeleteExpiredRefreshTokens(ctx)
}

func (r *RefreshTokenRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.RefreshToken, error) {
	dbTokens, err := r.queries.GetActiveRefreshTokensByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	tokens := make([]*entity.RefreshToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = dbRefreshTokenToEntity(dbToken)
	}
	return tokens, nil
}

func dbRefreshTokenToEntity(dbToken db.UsrRefreshToken) *entity.RefreshToken {
	token := &entity.RefreshToken{
		ID:        dbToken.ID,
		Token:     dbToken.Token,
		UserID:    dbToken.UserID,
		IsRevoked: dbToken.IsRevoked,
		CreatedAt: dbToken.CreatedAt.Time,
	}

	if dbToken.ExpiresAt.Valid {
		token.ExpiresAt = dbToken.ExpiresAt.Time
	}
	if dbToken.RevokedAt.Valid {
		t := dbToken.RevokedAt.Time
		token.RevokedAt = &t
	}

	return token
}
