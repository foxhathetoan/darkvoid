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

// HashtagRepository handles DB operations for hashtags.
type HashtagRepository struct {
	queries *db.Queries
	dbtx    db.DBTX       // underlying connection: pool (standalone) or tx (tx-scoped)
	pool    *pgxpool.Pool // nil when this instance is tx-scoped via WithTx
}

// NewHashtagRepository creates a new HashtagRepository.
func NewHashtagRepository(pool *pgxpool.Pool) *HashtagRepository {
	return &HashtagRepository{queries: db.New(pool), dbtx: pool, pool: pool}
}

// WithTx returns a new HashtagRepository that executes all operations within the given
// transaction. UpsertAndLink and ReplaceForPost will use the tx directly instead of
// opening a new one.
func (r *HashtagRepository) WithTx(tx pgx.Tx) *HashtagRepository {
	return &HashtagRepository{
		queries: r.queries.WithTx(tx),
		dbtx:    tx,
		pool:    nil, // nil signals: already inside an outer transaction
	}
}

// UpsertAndLink batch-upserts tag names then links all of them to postID.
// When called on a WithTx instance it joins the caller's transaction; otherwise it
// manages its own transaction.
func (r *HashtagRepository) UpsertAndLink(ctx context.Context, postID uuid.UUID, names []string) error {
	if len(names) == 0 {
		return nil
	}
	if r.pool == nil {
		// Already tx-scoped — execute within the caller's transaction.
		return r.upsertAndLinkTx(ctx, r.dbtx, postID, names)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return database.MapDBError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := r.upsertAndLinkTx(ctx, tx, postID, names); err != nil {
		return err
	}
	return database.MapDBError(tx.Commit(ctx))
}

// ReplaceForPost atomically deletes all existing hashtag links for a post and creates new ones.
// When called on a WithTx instance it joins the caller's transaction; otherwise it manages
// its own transaction.
func (r *HashtagRepository) ReplaceForPost(ctx context.Context, postID uuid.UUID, names []string) error {
	if r.pool == nil {
		// Already tx-scoped — execute within the caller's transaction.
		if err := database.MapDBError(r.queries.DeletePostHashtags(ctx, postID)); err != nil {
			return err
		}
		if len(names) > 0 {
			return r.upsertAndLinkTx(ctx, r.dbtx, postID, names)
		}
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return database.MapDBError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := database.MapDBError(r.queries.WithTx(tx).DeletePostHashtags(ctx, postID)); err != nil {
		return err
	}
	if len(names) > 0 {
		if err := r.upsertAndLinkTx(ctx, tx, postID, names); err != nil {
			return err
		}
	}
	return database.MapDBError(tx.Commit(ctx))
}

const sqlUpsertHashtagsBatch = `
INSERT INTO post.hashtags (name)
SELECT unnest($1::text[])
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id, name, created_at
`

const sqlLinkPostHashtagsBatch = `
INSERT INTO post.post_hashtags (post_id, hashtag_id)
SELECT $1::uuid, unnest($2::uuid[])
ON CONFLICT DO NOTHING
`

// upsertAndLinkTx batch-upserts hashtag names and links them to postID using the provided DBTX.
// Implemented directly on DBTX (not via *db.Queries) because sqlc cannot generate unnest-based batch inserts.
func (r *HashtagRepository) upsertAndLinkTx(ctx context.Context, dbtx db.DBTX, postID uuid.UUID, names []string) error {
	rows, err := dbtx.Query(ctx, sqlUpsertHashtagsBatch, names)
	if err != nil {
		return database.MapDBError(err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0, len(names))
	for rows.Next() {
		var h db.PostHashtag
		if err = rows.Scan(&h.ID, &h.Name, &h.CreatedAt); err != nil {
			return database.MapDBError(err)
		}
		ids = append(ids, h.ID)
	}
	if err = rows.Err(); err != nil {
		return database.MapDBError(err)
	}
	_, err = dbtx.Exec(ctx, sqlLinkPostHashtagsBatch, postID, ids)
	return database.MapDBError(err)
}

// GetNamesByPostID returns the hashtag names attached to a post.
func (r *HashtagRepository) GetNamesByPostID(ctx context.Context, postID uuid.UUID) ([]string, error) {
	rows, err := r.queries.GetHashtagsByPostID(ctx, postID)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	names := make([]string, len(rows))
	for i, h := range rows {
		names[i] = h.Name
	}
	return names, nil
}

// GetNamesByPostIDs batch-fetches hashtag names for multiple posts.
// Returns a map of postID → []string.
func (r *HashtagRepository) GetNamesByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	rows, err := r.queries.GetHashtagsByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make(map[uuid.UUID][]string, len(postIDs))
	for _, row := range rows {
		result[row.PostID] = append(result[row.PostID], row.Name)
	}
	return result, nil
}

// GetTrending returns the top N trending hashtags based on post usage in the last 24 hours.
func (r *HashtagRepository) GetTrending(ctx context.Context, limit int32) ([]*entity.TrendingHashtag, error) {
	rows, err := r.queries.GetTrendingHashtags(ctx, limit)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make([]*entity.TrendingHashtag, len(rows))
	for i, row := range rows {
		result[i] = &entity.TrendingHashtag{Name: row.Name, Count: row.Count}
	}
	return result, nil
}

// SearchByPrefix returns hashtag names that start with prefix, up to limit results.
func (r *HashtagRepository) SearchByPrefix(ctx context.Context, prefix string, limit int32) ([]string, error) {
	names, err := r.queries.SearchHashtagsByPrefix(ctx, db.SearchHashtagsByPrefixParams{Column1: prefix, Limit: limit})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return names, nil
}

// GetPostsByHashtag returns cursor-paginated posts for a given hashtag name.
func (r *HashtagRepository) GetPostsByHashtag(ctx context.Context, name string, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, limit int32) ([]*entity.Post, error) {
	rows, err := r.queries.GetPostsByHashtagWithCursor(ctx, db.GetPostsByHashtagWithCursorParams{
		Name:    name,
		Column2: cursorCreatedAt,
		Column3: cursorPostID,
		Limit:   limit,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return hashtagCursorRowsToPosts(rows), nil
}
