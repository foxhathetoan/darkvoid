package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/feature/post/repository"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// commentRepoTxable wraps *repository.CommentRepository so that its WithTx method can return
// the commentRepo interface without creating an import cycle.
type commentRepoTxable struct{ *repository.CommentRepository }

func (r *commentRepoTxable) WithTx(tx pgx.Tx) commentRepo {
	return &commentRepoTxable{r.CommentRepository.WithTx(tx)}
}

// commentMediaRepoTxable wraps *repository.CommentMediaRepository for the same reason.
type commentMediaRepoTxable struct {
	*repository.CommentMediaRepository
}

func (r *commentMediaRepoTxable) WithTx(tx pgx.Tx) commentMediaRepo {
	return &commentMediaRepoTxable{r.CommentMediaRepository.WithTx(tx)}
}

// CommentServiceOption is a functional option for configuring optional CommentService dependencies.
type CommentServiceOption func(*CommentService)

// WithCommentLikeRepo attaches a comment like repository for is-liked enrichment.
func WithCommentLikeRepo(r commentLikeRepo) CommentServiceOption {
	return func(s *CommentService) { s.commentLikeRepo = r }
}

// WithTrendingInvalidator wires a cross-context trending cache invalidator after construction.
// Called by the app layer once the feed cache is ready.
func (s *CommentService) WithTrendingInvalidator(inv TrendingInvalidator) {
	s.trendingInvalidator = inv
}

// WithNotificationEmitter wires a cross-context notification emitter after construction.
// Called by the app layer once the notification context is ready.
func (s *CommentService) WithNotificationEmitter(e CommentNotificationEmitter) {
	s.notifEmitter = e
}

// WithCommentMentionRepo attaches the comment mention repository.
func WithCommentMentionRepo(r commentMentionRepo) CommentServiceOption {
	return func(s *CommentService) { s.commentMentionRepo = r }
}

// CommentService handles comment business logic
type CommentService struct {
	pool                txBeginner
	commentRepo         commentRepo
	commentMediaRepo    commentMediaRepo
	postRepo            postRepo
	userReader          userReader
	commentLikeRepo     commentLikeRepo            // optional: nil → is-liked skipped
	trendingInvalidator TrendingInvalidator        // optional: nil → no-op
	notifEmitter        CommentNotificationEmitter // optional: nil → no-op
	commentMentionRepo  commentMentionRepo         // optional: nil → mentions skipped
}

// NewCommentService creates a new CommentService. Required dependencies are passed as positional
// arguments; optional ones are injected via CommentServiceOption functions.
func NewCommentService(
	pool *pgxpool.Pool,
	commentRepo *repository.CommentRepository,
	commentMediaRepo *repository.CommentMediaRepository,
	postRepo *repository.PostRepository,
	userReader userReader,
	opts ...CommentServiceOption,
) *CommentService {
	s := &CommentService{
		pool:             pool,
		commentRepo:      &commentRepoTxable{commentRepo},
		commentMediaRepo: &commentMediaRepoTxable{commentMediaRepo},
		postRepo:         &postRepoTxable{postRepo},
		userReader:       userReader,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *CommentService) invalidateTrending(ctx context.Context) {
	if s.trendingInvalidator == nil {
		return
	}
	if err := s.trendingInvalidator.InvalidateTrending(ctx); err != nil {
		logger.LogError(ctx, err, "failed to invalidate trending cache after comment change")
	}
}

// CreateComment adds a comment (or reply) to a post
func (s *CommentService) CreateComment(ctx context.Context, postID, authorID uuid.UUID, parentID *uuid.UUID, content string, mediaKeys []string, mentionUserIDs []uuid.UUID) (*entity.Comment, error) {
	if strings.TrimSpace(content) == "" && len(mediaKeys) == 0 {
		return nil, errors.NewBadRequestError("comment content or media is required")
	}
	p, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, post.ErrPostNotFound
		}
		return nil, err
	}
	// Validate parent comment belongs to same post
	var parent *entity.Comment
	if parentID != nil {
		parent, err = s.commentRepo.GetByID(ctx, *parentID)
		if err != nil {
			if errors.Is(err, errors.ErrNotFound) {
				return nil, post.ErrCommentNotFound
			}
			return nil, err
		}
		if parent.PostID != postID {
			return nil, post.ErrCommentNotFound
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	txComment := s.commentRepo.WithTx(tx)
	txMedia := s.commentMediaRepo.WithTx(tx)

	c, err := txComment.Create(ctx, postID, authorID, parentID, strings.TrimSpace(content))
	if err != nil {
		logger.LogError(ctx, err, "failed to create comment", "post_id", postID, "author_id", authorID)
		return nil, errors.NewInternalError(err)
	}

	for i, key := range mediaKeys {
		mediaType := inferMediaType(key)
		media, err := txMedia.Add(ctx, c.ID, key, mediaType, int32(i))
		if err != nil {
			logger.LogError(ctx, err, "failed to attach media to comment", "comment_id", c.ID)
			return nil, errors.NewInternalError(err)
		}
		c.Media = append(c.Media, media)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.NewInternalError(err)
	}

	s.enrichAuthors(ctx, []*entity.Comment{c})
	s.invalidateTrending(ctx)

	// Persist mentions — non-fatal side-effect
	c.Mentions = s.persistCommentMentions(ctx, c.ID, authorID, mentionUserIDs)

	// Emit notification: reply → parent comment author, top-level comment → post author
	if parent != nil {
		s.emitReplyNotification(ctx, authorID, parent.AuthorID, *parentID, c.ID)
	} else {
		s.emitCommentNotification(ctx, authorID, p.AuthorID, postID, c.ID)
	}

	logger.Info(ctx, "comment created", "comment_id", c.ID, "post_id", postID)
	return c, nil
}

// GetComments returns top-level paginated comments for a post
func (s *CommentService) GetComments(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
	req.Validate()

	if _, err := s.postRepo.GetByID(ctx, postID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, pagination.PaginationResponse{}, post.ErrPostNotFound
		}
		return nil, pagination.PaginationResponse{}, err
	}

	comments, err := s.commentRepo.GetByPost(ctx, postID, req.Limit, req.Offset)
	if err != nil {
		logger.LogError(ctx, err, "failed to get comments", "post_id", postID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}
	total, err := s.commentRepo.CountByPost(ctx, postID)
	if err != nil {
		logger.LogError(ctx, err, "failed to count comments", "post_id", postID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}

	s.enrichAuthors(ctx, comments)
	s.enrichMedia(ctx, comments)
	s.enrichLikes(ctx, comments, viewerID)
	s.enrichCommentMentions(ctx, comments)
	s.attachReplyPreviews(ctx, comments, viewerID)
	s.attachReplyCounts(ctx, comments)
	return comments, pagination.NewPaginationResponse(total, req.Limit, req.Offset), nil
}

// GetReplies returns paginated replies for a comment
func (s *CommentService) GetReplies(ctx context.Context, commentID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error) {
	req.Validate()

	if _, err := s.commentRepo.GetByID(ctx, commentID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, pagination.PaginationResponse{}, post.ErrCommentNotFound
		}
		return nil, pagination.PaginationResponse{}, err
	}

	replies, err := s.commentRepo.GetReplies(ctx, commentID, req.Limit, req.Offset)
	if err != nil {
		logger.LogError(ctx, err, "failed to get replies", "comment_id", commentID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}

	s.enrichAuthors(ctx, replies)
	s.enrichMedia(ctx, replies)
	s.enrichLikes(ctx, replies, viewerID)
	s.enrichCommentMentions(ctx, replies)
	s.attachReplyCounts(ctx, replies)

	counts, err := s.commentRepo.GetReplyCountsBatch(ctx, []uuid.UUID{commentID})
	total := int64(0)
	if err != nil {
		logger.LogError(ctx, err, "failed to count replies", "comment_id", commentID)
	} else {
		total = counts[commentID]
	}
	return replies, pagination.NewPaginationResponse(total, req.Limit, req.Offset), nil
}

// attachReplyCounts batch-fetches total descendant counts for each comment.
func (s *CommentService) attachReplyCounts(ctx context.Context, comments []*entity.Comment) {
	if len(comments) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}
	counts, err := s.commentRepo.GetReplyCountsBatch(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to fetch reply counts")
		return
	}
	for _, c := range comments {
		c.ReplyCount = counts[c.ID]
	}
}

// attachReplyPreviews batch-fetches up to 2 replies for each comment and attaches them.
func (s *CommentService) attachReplyPreviews(ctx context.Context, comments []*entity.Comment, viewerID *uuid.UUID) {
	if len(comments) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}
	repliesMap, err := s.commentRepo.GetRepliesPreview(ctx, ids, 2)
	if err != nil {
		logger.LogError(ctx, err, "failed to fetch reply previews")
		return
	}
	var allReplies []*entity.Comment
	for _, c := range comments {
		if replies, ok := repliesMap[c.ID]; ok {
			c.Replies = replies
			allReplies = append(allReplies, replies...)
		}
	}
	s.enrichAuthors(ctx, allReplies)
	s.enrichMedia(ctx, allReplies)
	s.enrichLikes(ctx, allReplies, viewerID)
	s.enrichCommentMentions(ctx, allReplies)
	s.attachReplyCounts(ctx, allReplies)
}

// enrichLikes batch-fetches is_liked for a slice of comments for the given viewer.
func (s *CommentService) enrichLikes(ctx context.Context, comments []*entity.Comment, viewerID *uuid.UUID) {
	if len(comments) == 0 || viewerID == nil || s.commentLikeRepo == nil {
		return
	}
	ids := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}
	likedIDs, err := s.commentLikeRepo.GetLikedCommentIDs(ctx, *viewerID, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich comment likes")
		return
	}
	liked := make(map[uuid.UUID]bool, len(likedIDs))
	for _, id := range likedIDs {
		liked[id] = true
	}
	for _, c := range comments {
		c.IsLiked = liked[c.ID]
	}
}

// enrichMedia batch-fetches media attachments for a slice of comments.
func (s *CommentService) enrichMedia(ctx context.Context, comments []*entity.Comment) {
	if len(comments) == 0 || s.commentMediaRepo == nil {
		return
	}
	ids := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}
	mediaMap, err := s.commentMediaRepo.GetByCommentsBatch(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich comment media")
		return
	}
	for _, c := range comments {
		if m, ok := mediaMap[c.ID]; ok {
			c.Media = m
		}
	}
}

// enrichAuthors batch-fetches author info for a slice of comments.
func (s *CommentService) enrichAuthors(ctx context.Context, comments []*entity.Comment) {
	if len(comments) == 0 || s.userReader == nil {
		return
	}
	seen := make(map[uuid.UUID]bool, len(comments))
	ids := make([]uuid.UUID, 0, len(comments))
	for _, c := range comments {
		if !seen[c.AuthorID] {
			seen[c.AuthorID] = true
			ids = append(ids, c.AuthorID)
		}
	}
	authors, err := s.userReader.GetAuthorsByIDs(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich comment authors")
		return
	}
	for _, c := range comments {
		if a, ok := authors[c.AuthorID]; ok {
			c.Author = a
		}
	}
}

// persistCommentMentions inserts mention rows for the given user IDs, enriches them with author
// info, and fires EmitMention for each. Returns the enriched MentionedUser slice.
// Non-fatal — logs errors and continues.
func (s *CommentService) persistCommentMentions(ctx context.Context, commentID, actorID uuid.UUID, mentionIDs []uuid.UUID) []*entity.MentionedUser {
	if s.commentMentionRepo == nil || len(mentionIDs) == 0 {
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
		logger.LogError(ctx, err, "failed to enrich mention authors", "comment_id", commentID)
		authors = make(map[uuid.UUID]*entity.Author)
	}

	mentioned := make([]*entity.MentionedUser, 0, len(ids))
	for _, uid := range ids {
		if err := s.commentMentionRepo.Insert(ctx, commentID, uid); err != nil {
			logger.LogError(ctx, err, "failed to insert comment mention", "comment_id", commentID, "user_id", uid)
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
			if err := s.notifEmitter.EmitMention(ctx, actorID, uid, commentID); err != nil {
				logger.LogError(ctx, err, "failed to emit mention notification", "comment_id", commentID, "recipient_id", uid)
			}
		}
	}
	return mentioned
}

// enrichCommentMentions batch-loads mention user info for a slice of comments.
func (s *CommentService) enrichCommentMentions(ctx context.Context, comments []*entity.Comment) {
	if s.commentMentionRepo == nil || len(comments) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}
	mentionMap, err := s.commentMentionRepo.GetBatch(ctx, ids)
	if err != nil {
		logger.LogError(ctx, err, "failed to batch fetch comment mentions")
		return
	}

	// Collect all unique user IDs across all comments
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
		logger.LogError(ctx, err, "failed to enrich comment mention authors")
		return
	}

	for _, c := range comments {
		userIDs, ok := mentionMap[c.ID]
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
		c.Mentions = mentions
	}
}

// --- notification helpers (fire-and-forget) ---

func (s *CommentService) emitCommentNotification(ctx context.Context, actorID, recipientID, postID, commentID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.EmitComment(ctx, actorID, recipientID, postID, commentID); err != nil {
		logger.LogError(ctx, err, "failed to emit comment notification", "actor", actorID, "post", postID)
	}
}

func (s *CommentService) emitReplyNotification(ctx context.Context, actorID, recipientID, parentCommentID, replyID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.EmitReply(ctx, actorID, recipientID, parentCommentID, replyID); err != nil {
		logger.LogError(ctx, err, "failed to emit reply notification", "actor", actorID, "parent_comment", parentCommentID)
	}
}

// DeleteComment soft-deletes a comment (only by its author)
func (s *CommentService) DeleteComment(ctx context.Context, commentID, userID uuid.UUID) error {
	c, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return post.ErrCommentNotFound
		}
		return err
	}
	if c.AuthorID != userID {
		return post.ErrForbidden
	}
	if err := s.commentRepo.Delete(ctx, commentID); err != nil {
		logger.LogError(ctx, err, "failed to delete comment", "comment_id", commentID)
		return errors.NewInternalError(err)
	}
	s.invalidateTrending(ctx)
	logger.Info(ctx, "comment deleted", "comment_id", commentID)
	return nil
}
