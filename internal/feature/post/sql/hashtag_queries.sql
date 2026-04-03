-- name: UpsertHashtag :one
INSERT INTO post.hashtags (name)
VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id, name, created_at;

-- name: LinkPostHashtag :exec
INSERT INTO post.post_hashtags (post_id, hashtag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- Hand-patched batch queries (sqlc cannot generate unnest-based batch inserts):
--
-- UpsertHashtagsBatch: batch upsert hashtag names, returns all upserted rows
--   INSERT INTO post.hashtags (name) SELECT unnest($1::text[])
--   ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
--   RETURNING id, name, created_at
--
-- LinkPostHashtagsBatch: link a post to multiple hashtag IDs in one statement
--   INSERT INTO post.post_hashtags (post_id, hashtag_id)
--   SELECT $1::uuid, unnest($2::uuid[])
--   ON CONFLICT DO NOTHING

-- name: DeletePostHashtags :exec
DELETE FROM post.post_hashtags WHERE post_id = $1;

-- name: GetHashtagsByPostID :many
SELECT h.id, h.name, h.created_at
FROM post.hashtags h
JOIN post.post_hashtags ph ON ph.hashtag_id = h.id
WHERE ph.post_id = $1
ORDER BY h.name;

-- name: GetHashtagsByPostIDs :many
SELECT ph.post_id, h.name
FROM post.hashtags h
JOIN post.post_hashtags ph ON ph.hashtag_id = h.id
WHERE ph.post_id = ANY($1::uuid[])
ORDER BY ph.post_id, h.name;

-- name: GetTrendingHashtags :many
SELECT h.name, COUNT(*) AS count
FROM post.post_hashtags ph
JOIN post.hashtags h  ON h.id  = ph.hashtag_id
JOIN post.posts    p  ON p.id  = ph.post_id
WHERE p.created_at > NOW() - INTERVAL '24 hours'
  AND p.visibility = 'public'
  AND p.deleted_at IS NULL
GROUP BY h.id, h.name
ORDER BY count DESC
LIMIT $1;

-- name: GetPostsByHashtagWithCursor :many
SELECT p.id, p.author_id, p.content, p.visibility, p.created_at, p.updated_at, p.deleted_at, p.like_count, p.comment_count
FROM post.posts p
JOIN post.post_hashtags ph ON ph.post_id   = p.id
JOIN post.hashtags      h  ON h.id         = ph.hashtag_id
WHERE h.name         = $1
  AND p.visibility   = 'public'
  AND p.deleted_at   IS NULL
  AND (p.created_at < $2::timestamptz
       OR (p.created_at = $2::timestamptz AND p.id < $3::uuid))
ORDER BY p.created_at DESC, p.id DESC
LIMIT $4;

-- name: SearchHashtagsByPrefix :many
SELECT name
FROM post.hashtags
WHERE name LIKE $1::text || '%'
ORDER BY name
LIMIT $2;
