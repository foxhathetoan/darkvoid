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

type MediaRepository struct {
	queries *db.Queries
	dbtx    db.DBTX
}

func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{queries: db.New(pool), dbtx: pool}
}

// WithTx returns a new MediaRepository that executes queries within the given transaction.
func (r *MediaRepository) WithTx(tx pgx.Tx) *MediaRepository {
	return &MediaRepository{queries: r.queries.WithTx(tx), dbtx: tx}
}

func (r *MediaRepository) Add(ctx context.Context, postID uuid.UUID, mediaKey, mediaType string, position int32) (*entity.PostMedia, error) {
	row, err := r.queries.AddPostMedia(ctx, db.AddPostMediaParams{
		PostID:    postID,
		MediaKey:  mediaKey,
		MediaType: mediaType,
		Position:  position,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return rowToMedia(row), nil
}

func (r *MediaRepository) GetByPost(ctx context.Context, postID uuid.UUID) ([]*entity.PostMedia, error) {
	rows, err := r.queries.GetPostMedia(ctx, postID)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	result := make([]*entity.PostMedia, len(rows))
	for i, row := range rows {
		result[i] = rowToMedia(row)
	}
	return result, nil
}

func (r *MediaRepository) Delete(ctx context.Context, id, postID uuid.UUID) error {
	return database.MapDBError(r.queries.DeletePostMedia(ctx, db.DeletePostMediaParams{
		ID:     id,
		PostID: postID,
	}))
}

func (r *MediaRepository) DeleteAllByPost(ctx context.Context, postID uuid.UUID) error {
	return database.MapDBError(r.queries.DeleteAllPostMedia(ctx, postID))
}

// GetByPostsBatch fetches all media for the given post IDs in a single query.
func (r *MediaRepository) GetByPostsBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]*entity.PostMedia, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID][]*entity.PostMedia), nil
	}
	rows, err := r.dbtx.Query(ctx,
		`SELECT id, post_id, media_key, media_type, position, created_at
		 FROM post.post_media
		 WHERE post_id = ANY($1::uuid[])
		 ORDER BY post_id, position ASC`,
		postIDs)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	defer rows.Close()
	result := make(map[uuid.UUID][]*entity.PostMedia, len(postIDs))
	for rows.Next() {
		var m db.PostPostMedium
		if err := rows.Scan(&m.ID, &m.PostID, &m.MediaKey, &m.MediaType, &m.Position, &m.CreatedAt); err != nil {
			return nil, database.MapDBError(err)
		}
		media := rowToMedia(m)
		result[m.PostID] = append(result[m.PostID], media)
	}
	if err := rows.Err(); err != nil {
		return nil, database.MapDBError(err)
	}
	return result, nil
}

func rowToMedia(row db.PostPostMedium) *entity.PostMedia {
	return &entity.PostMedia{
		ID:        row.ID,
		PostID:    row.PostID,
		MediaKey:  row.MediaKey,
		MediaType: row.MediaType,
		Position:  row.Position,
		CreatedAt: row.CreatedAt.Time,
	}
}
