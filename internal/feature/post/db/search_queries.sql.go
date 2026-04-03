package db

import (
	"context"
)

const searchPosts = `-- name: SearchPosts :many
SELECT id, author_id, content, visibility, created_at, updated_at, deleted_at, like_count, comment_count
FROM post.posts
WHERE deleted_at IS NULL
  AND visibility = 'public'
  AND search_vector @@ plainto_tsquery('english', $1)
ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC, created_at DESC
LIMIT $2 OFFSET $3
`

type SearchPostsParams struct {
	Query  string `json:"query"`
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
}

// SearchPosts performs a full-text search over public posts using tsvector.
func (q *Queries) SearchPosts(ctx context.Context, arg SearchPostsParams) ([]PostPost, error) {
	rows, err := q.db.Query(ctx, searchPosts, arg.Query, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PostPost{}
	for rows.Next() {
		var i PostPost
		if err := rows.Scan(
			&i.ID,
			&i.AuthorID,
			&i.Content,
			&i.Visibility,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
			&i.LikeCount,
			&i.CommentCount,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
