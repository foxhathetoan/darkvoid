package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/feature/post/repository"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// tagNameRegex validates an individual tag name (alphanumeric + underscore, 1–50 chars).
var tagNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,50}$`)

// postRepoTxable wraps *repository.PostRepository so that its WithTx method can return
// the postRepo interface (defined in this package) without creating an import cycle.
type postRepoTxable struct{ *repository.PostRepository }

func (r *postRepoTxable) WithTx(tx pgx.Tx) postRepo {
	return &postRepoTxable{r.PostRepository.WithTx(tx)}
}

// mediaRepoTxable wraps *repository.MediaRepository for the same reason.
type mediaRepoTxable struct{ *repository.MediaRepository }

func (r *mediaRepoTxable) WithTx(tx pgx.Tx) mediaRepo {
	return &mediaRepoTxable{r.MediaRepository.WithTx(tx)}
}

// hashtagRepoTxable wraps *repository.HashtagRepository so that its WithTx method can return
// the hashtagRepo interface without creating an import cycle.
type hashtagRepoTxable struct{ *repository.HashtagRepository }

func (r *hashtagRepoTxable) WithTx(tx pgx.Tx) hashtagRepo {
	return &hashtagRepoTxable{r.HashtagRepository.WithTx(tx)}
}

// PostServiceOption is a functional option for configuring optional PostService dependencies.
type PostServiceOption func(*PostService)

// WithLikeRepo attaches a like repository for like-count and is-liked enrichment.
func WithLikeRepo(r likeRepo) PostServiceOption {
	return func(s *PostService) { s.likeRepo = r }
}

// WithMentionRepo attaches the mention repository.
func WithMentionRepo(r mentionRepo) PostServiceOption {
	return func(s *PostService) { s.mentionRepo = r }
}

// WithNotificationEmitter wires a cross-context notification emitter after construction.
// Called by the app layer once the notification context is ready.
func (s *PostService) WithNotificationEmitter(e notificationEmitter) {
	s.notifEmitter = e
}

// PostService handles post business logic
type PostService struct {
	pool        txBeginner
	postRepo    postRepo
	mediaRepo   mediaRepo
	likeRepo    likeRepo // optional: nil → like count/isLiked skipped
	userReader  userReader
	hashtagRepo hashtagRepo
	mentionRepo mentionRepo // optional: nil → mentions skipped

	notifEmitter notificationEmitter // optional: nil → notifications skipped
}

// NewPostService creates a new PostService. Required dependencies are passed as positional
// arguments; optional ones are injected via PostServiceOption functions.
func NewPostService(
	pool *pgxpool.Pool,
	postRepo *repository.PostRepository,
	mediaRepo *repository.MediaRepository,
	userReader userReader,
	hashtagRepo *repository.HashtagRepository,
	opts ...PostServiceOption,
) *PostService {
	s := &PostService{
		pool:        pool,
		postRepo:    &postRepoTxable{postRepo},
		mediaRepo:   &mediaRepoTxable{mediaRepo},
		userReader:  userReader,
		hashtagRepo: &hashtagRepoTxable{hashtagRepo},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreatePost creates a new post
func (s *PostService) CreatePost(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility, mediaKeys []string, mentionUserIDs []uuid.UUID, tags []string) (*entity.Post, error) {
	if strings.TrimSpace(content) == "" && len(mediaKeys) == 0 {
		return nil, post.ErrEmptyContent
	}
	if !isValidVisibility(visibility) {
		return nil, post.ErrInvalidVisibility
	}

	validTags, err := validateTags(tags)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	txPost := s.postRepo.WithTx(tx)
	txMedia := s.mediaRepo.WithTx(tx)

	p, err := txPost.Create(ctx, authorID, strings.TrimSpace(content), visibility)
	if err != nil {
		logger.LogError(ctx, err, "failed to create post", "author_id", authorID)
		return nil, errors.NewInternalError(err)
	}

	for i, key := range mediaKeys {
		media, err := txMedia.Add(ctx, p.ID, key, inferMediaType(key), int32(i))
		if err != nil {
			logger.LogError(ctx, err, "failed to attach media", "post_id", p.ID)
			return nil, errors.NewInternalError(err)
		}
		p.Media = append(p.Media, media)
	}

	if len(validTags) > 0 && s.hashtagRepo != nil {
		if err := s.hashtagRepo.WithTx(tx).UpsertAndLink(ctx, p.ID, validTags); err != nil {
			logger.LogError(ctx, err, "failed to persist hashtags", "post_id", p.ID)
			return nil, errors.NewInternalError(err)
		}
		p.Tags = validTags
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.NewInternalError(err)
	}

	// Persist mentions — non-fatal side-effect
	p.Mentions = s.persistMentions(ctx, p.ID, authorID, mentionUserIDs)

	logger.Info(ctx, "post created", "post_id", p.ID, "author_id", authorID)
	return p, nil
}

// GetPost retrieves a single post by ID, enriched with like count and optional isLiked flag
func (s *PostService) GetPost(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID) (*entity.Post, error) {
	p, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, post.ErrPostNotFound
		}
		return nil, err
	}

	s.enrichBatch(ctx, []*entity.Post{p}, viewerID)
	s.enrichAuthors(ctx, []*entity.Post{p})
	s.enrichTags(ctx, []*entity.Post{p})
	s.enrichMentions(ctx, []*entity.Post{p})
	return p, nil
}

// GetUserPosts returns cursor-paginated posts for a user, optionally filtered by visibility.
// cursor nil means start from the latest post. visibility "" means no filter.
func (s *PostService) GetUserPosts(ctx context.Context, authorID uuid.UUID, viewerID *uuid.UUID, cursor *post.UserPostCursor, visibility string, limit int32) ([]*entity.Post, *post.UserPostCursor, error) {
	if limit <= 0 {
		limit = 20
	}

	cursorTS := pgtype.Timestamptz{Time: post.MaxUserPostTime, Valid: true}
	cursorID := uuid.Max
	if cursor != nil {
		cursorTS = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		var err error
		cursorID, err = uuid.Parse(cursor.PostID)
		if err != nil {
			return nil, nil, errors.NewBadRequestError("invalid cursor post_id")
		}
	}

	// Fetch one extra to detect if there's a next page
	posts, err := s.postRepo.GetByAuthorWithCursor(ctx, authorID, cursorTS, cursorID, visibility, limit+1)
	if err != nil {
		logger.LogError(ctx, err, "failed to get user posts", "author_id", authorID)
		return nil, nil, errors.NewInternalError(err)
	}

	var nextCursor *post.UserPostCursor
	if len(posts) > int(limit) {
		last := posts[limit-1]
		nextCursor = &post.UserPostCursor{
			CreatedAt: last.CreatedAt,
			PostID:    last.ID.String(),
		}
		posts = posts[:limit]
	}

	s.enrichBatch(ctx, posts, viewerID)
	s.enrichAuthors(ctx, posts)
	s.enrichTags(ctx, posts)
	s.enrichMentions(ctx, posts)

	return posts, nextCursor, nil
}

// UpdatePost updates content/visibility of a post (only by owner)
func (s *PostService) UpdatePost(ctx context.Context, postID, userID uuid.UUID, content string, visibility entity.Visibility, mentionUserIDs []uuid.UUID, tags []string) (*entity.Post, error) {
	existing, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, post.ErrPostNotFound
		}
		return nil, err
	}
	if existing.AuthorID != userID {
		return nil, post.ErrForbidden
	}
	if !isValidVisibility(visibility) {
		return nil, post.ErrInvalidVisibility
	}

	validTags, err := validateTags(tags)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	txPost := s.postRepo.WithTx(tx)

	updated, err := txPost.Update(ctx, postID, strings.TrimSpace(content), visibility)
	if err != nil {
		logger.LogError(ctx, err, "failed to update post", "post_id", postID)
		return nil, errors.NewInternalError(err)
	}

	if s.hashtagRepo != nil {
		if err := s.hashtagRepo.WithTx(tx).ReplaceForPost(ctx, postID, validTags); err != nil {
			logger.LogError(ctx, err, "failed to replace hashtags", "post_id", postID)
			return nil, errors.NewInternalError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.NewInternalError(err)
	}

	// Replace mentions — non-fatal side-effect, outside tx
	if s.mentionRepo != nil {
		if err := s.mentionRepo.DeleteByPost(ctx, postID); err != nil {
			logger.LogError(ctx, err, "failed to clear old mentions", "post_id", postID)
		}
	}
	updated.Mentions = s.persistMentions(ctx, postID, userID, mentionUserIDs)

	s.enrichBatch(ctx, []*entity.Post{updated}, &userID)
	s.enrichAuthors(ctx, []*entity.Post{updated})
	s.enrichTags(ctx, []*entity.Post{updated})
	s.enrichMentions(ctx, []*entity.Post{updated})
	logger.Info(ctx, "post updated", "post_id", postID)
	return updated, nil
}

// DeletePost soft-deletes a post (only by owner)
func (s *PostService) DeletePost(ctx context.Context, postID, userID uuid.UUID) error {
	existing, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return post.ErrPostNotFound
		}
		return err
	}
	if existing.AuthorID != userID {
		return post.ErrForbidden
	}
	if err := s.postRepo.Delete(ctx, postID); err != nil {
		logger.LogError(ctx, err, "failed to delete post", "post_id", postID)
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "post deleted", "post_id", postID)
	return nil
}

// enrichBatch loads media and isLiked for a slice of posts in bulk.
// LikeCount is already populated from the denormalized DB counter on each post row.
// Like-related fields are skipped when likeRepo is not configured.
func (s *PostService) enrichBatch(ctx context.Context, posts []*entity.Post, viewerID *uuid.UUID) {
	if len(posts) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	mediaMap, err := s.mediaRepo.GetByPostsBatch(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to batch fetch post media")
	} else {
		for _, p := range posts {
			if media, ok := mediaMap[p.ID]; ok {
				p.Media = media
			}
		}
	}

	if s.likeRepo != nil && viewerID != nil {
		likedIDs, err := s.likeRepo.GetLikedPostIDs(ctx, *viewerID, ids)
		if err != nil {
			logger.LogError(ctx, err, "failed to batch fetch liked post IDs")
		} else {
			likedSet := make(map[uuid.UUID]bool, len(likedIDs))
			for _, id := range likedIDs {
				likedSet[id] = true
			}
			for _, p := range posts {
				p.IsLiked = likedSet[p.ID]
			}
		}
	}
}

// enrichAuthors batch-fetches author info for a slice of posts.
func (s *PostService) enrichAuthors(ctx context.Context, posts []*entity.Post) {
	if len(posts) == 0 || s.userReader == nil {
		return
	}
	seen := make(map[uuid.UUID]bool, len(posts))
	ids := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		if !seen[p.AuthorID] {
			seen[p.AuthorID] = true
			ids = append(ids, p.AuthorID)
		}
	}
	authors, err := s.userReader.GetAuthorsByIDs(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich post authors")
		return
	}
	for _, p := range posts {
		if a, ok := authors[p.AuthorID]; ok {
			p.Author = a
		}
	}
}

// persistMentions inserts mention rows for the given user IDs, enriches them with author info,
// and fires EmitMention for each. Returns the enriched MentionedUser slice.
// Non-fatal — logs errors and continues.
func (s *PostService) persistMentions(ctx context.Context, postID, actorID uuid.UUID, mentionIDs []uuid.UUID) []*entity.MentionedUser {
	if s.mentionRepo == nil || len(mentionIDs) == 0 {
		return nil
	}

	// Deduplicate
	seen := make(map[uuid.UUID]struct{}, len(mentionIDs))
	ids := make([]uuid.UUID, 0, len(mentionIDs))
	for _, uid := range mentionIDs {
		if _, ok := seen[uid]; !ok {
			seen[uid] = struct{}{}
			ids = append(ids, uid)
		}
	}

	authors, err := s.userReader.GetAuthorsByIDs(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich mention authors", "post_id", postID)
		authors = make(map[uuid.UUID]*entity.Author)
	}

	mentioned := make([]*entity.MentionedUser, 0, len(ids))
	for _, uid := range ids {
		if err := s.mentionRepo.Insert(ctx, postID, uid); err != nil {
			logger.LogError(ctx, err, "failed to insert mention", "post_id", postID, "user_id", uid)
			continue
		}
		if a, ok := authors[uid]; ok {
			mentioned = append(mentioned, &entity.MentionedUser{
				ID:          a.ID,
				Username:    a.Username,
				DisplayName: a.DisplayName,
			})
		}
		if s.notifEmitter != nil {
			if err := s.notifEmitter.EmitMention(ctx, actorID, uid, postID); err != nil {
				logger.LogError(ctx, err, "failed to emit mention notification", "post_id", postID, "recipient_id", uid)
			}
		}
	}
	return mentioned
}

// enrichMentions batch-loads mention user info for a slice of posts.
func (s *PostService) enrichMentions(ctx context.Context, posts []*entity.Post) {
	if s.mentionRepo == nil || len(posts) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	mentionMap, err := s.mentionRepo.GetBatch(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to batch fetch post mentions")
		return
	}

	// Collect all unique user IDs across all posts
	seen := make(map[uuid.UUID]bool)
	for _, userIDs := range mentionMap {
		for _, uid := range userIDs {
			seen[uid] = true
		}
	}
	if len(seen) == 0 {
		return
	}
	allIDs := make([]uuid.UUID, 0, len(seen))
	for uid := range seen {
		allIDs = append(allIDs, uid)
	}
	authors, err := s.userReader.GetAuthorsByIDs(ctx, allIDs)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich post mention authors")
		return
	}

	for _, p := range posts {
		userIDs, ok := mentionMap[p.ID]
		if !ok {
			continue
		}
		mentions := make([]*entity.MentionedUser, 0, len(userIDs))
		for _, uid := range userIDs {
			if a, ok := authors[uid]; ok {
				mentions = append(mentions, &entity.MentionedUser{
					ID:          a.ID,
					Username:    a.Username,
					DisplayName: a.DisplayName,
				})
			}
		}
		p.Mentions = mentions
	}
}

// validateTags normalizes and validates an explicit tag list from the client.
// Returns a deduplicated, lowercase slice or an error if any tag is invalid.
func validateTags(tags []string) ([]string, error) {
	if len(tags) > 10 {
		return nil, errors.NewBadRequestError("max 10 tags per post")
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if !tagNameRegex.MatchString(t) {
			return nil, errors.NewBadRequestError("invalid tag: " + t)
		}
		if _, dup := seen[t]; !dup {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out, nil
}

// enrichTags batch-fetches hashtag names for a slice of posts.
func (s *PostService) enrichTags(ctx context.Context, posts []*entity.Post) {
	if len(posts) == 0 || s.hashtagRepo == nil {
		return
	}
	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	tagsMap, err := s.hashtagRepo.GetNamesByPostIDs(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich post tags")
		return
	}
	for _, p := range posts {
		if names, ok := tagsMap[p.ID]; ok {
			p.Tags = names
		}
	}
}

func isValidVisibility(v entity.Visibility) bool {
	return v == entity.VisibilityPublic || v == entity.VisibilityFollowers || v == entity.VisibilityPrivate
}

func inferMediaType(key string) string {
	lower := strings.ToLower(key)
	if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".webm") {
		return "video"
	}
	return "image"
}
