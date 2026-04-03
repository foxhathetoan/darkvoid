package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
)

// EmailTokenRepository handles email token persistence via sqlc-generated queries.
type EmailTokenRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewEmailTokenRepository creates a new EmailTokenRepository.
func NewEmailTokenRepository(pool *pgxpool.Pool) *EmailTokenRepository {
	return &EmailTokenRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Create inserts a new email token and returns it.
func (r *EmailTokenRepository) Create(ctx context.Context, userID uuid.UUID, token string, tokenType entity.EmailTokenType, expiresAt time.Time) (*entity.EmailToken, error) {
	row, err := r.queries.CreateEmailToken(ctx, db.CreateEmailTokenParams{
		UserID:    userID,
		Token:     token,
		Type:      string(tokenType),
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbEmailTokenToEntity(row), nil
}

// GetByToken retrieves a token by its value.
func (r *EmailTokenRepository) GetByToken(ctx context.Context, token string) (*entity.EmailToken, error) {
	row, err := r.queries.GetEmailTokenByToken(ctx, token)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbEmailTokenToEntity(row), nil
}

// MarkUsed sets the used_at timestamp on a token, preventing reuse.
func (r *EmailTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	return database.MapDBError(r.queries.MarkEmailTokenUsed(ctx, id))
}

// DeleteByUserAndType removes all tokens of a given type for a user.
func (r *EmailTokenRepository) DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType entity.EmailTokenType) error {
	return database.MapDBError(r.queries.DeleteEmailTokensByUserAndType(ctx, db.DeleteEmailTokensByUserAndTypeParams{
		UserID: userID,
		Type:   string(tokenType),
	}))
}

// DeleteExpired removes all expired tokens.
func (r *EmailTokenRepository) DeleteExpired(ctx context.Context) error {
	return database.MapDBError(r.queries.DeleteExpiredEmailTokens(ctx))
}

func dbEmailTokenToEntity(row db.UsrEmailToken) *entity.EmailToken {
	et := &entity.EmailToken{
		ID:        row.ID,
		UserID:    row.UserID,
		Token:     row.Token,
		Type:      entity.EmailTokenType(row.Type),
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.UsedAt.Valid {
		t := row.UsedAt.Time
		et.UsedAt = &t
	}
	return et
}
