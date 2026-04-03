package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	postcache "github.com/jarviisha/darkvoid/internal/feature/post/cache"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
)

// txBeginner abstracts pgxpool.Pool.Begin so it can be mocked in tests.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// postRepo defines the repository operations needed by PostService and related services.
// *postRepoTxable satisfies this interface in production; *mockPostRepo in tests.
type postRepo interface {
	WithTx(tx pgx.Tx) postRepo
	Create(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Post, error)
	GetByAuthorWithCursor(ctx context.Context, authorID uuid.UUID, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, visibilityFilter string, limit int32) ([]*entity.Post, error)
	Update(ctx context.Context, id uuid.UUID, content string, visibility entity.Visibility) (*entity.Post, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// mediaRepo defines the repository operations needed by PostService.
// *mediaRepoTxable satisfies this interface in production; *mockMediaRepo in tests.
type mediaRepo interface {
	WithTx(tx pgx.Tx) mediaRepo
	Add(ctx context.Context, postID uuid.UUID, key, mediaType string, position int32) (*entity.PostMedia, error)
	GetByPost(ctx context.Context, postID uuid.UUID) ([]*entity.PostMedia, error)
	GetByPostsBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]*entity.PostMedia, error)
}

// likeRepo defines the repository operations needed by PostService and LikeService.
// *repository.LikeRepository satisfies this interface.
type likeRepo interface {
	Like(ctx context.Context, userID, postID uuid.UUID) error
	Unlike(ctx context.Context, userID, postID uuid.UUID) error
	IsLiked(ctx context.Context, userID, postID uuid.UUID) (bool, error)
	Count(ctx context.Context, postID uuid.UUID) (int64, error)
	GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) ([]uuid.UUID, error)
}

// commentRepo defines the repository operations needed by CommentService.
// *commentRepoTxable satisfies this interface in production; *mockCommentRepo in tests.
type commentRepo interface {
	WithTx(tx pgx.Tx) commentRepo
	Create(ctx context.Context, postID, authorID uuid.UUID, parentID *uuid.UUID, content string) (*entity.Comment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Comment, error)
	GetByPost(ctx context.Context, postID uuid.UUID, limit, offset int32) ([]*entity.Comment, error)
	CountByPost(ctx context.Context, postID uuid.UUID) (int64, error)
	GetReplies(ctx context.Context, parentID uuid.UUID, limit, offset int32) ([]*entity.Comment, error)
	GetRepliesPreview(ctx context.Context, parentIDs []uuid.UUID, limitPerParent int32) (map[uuid.UUID][]*entity.Comment, error)
	GetReplyCountsBatch(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// commentMediaRepo defines the repository operations needed by CommentService for media.
// *commentMediaRepoTxable satisfies this interface in production; *mockCommentMediaRepo in tests.
type commentMediaRepo interface {
	WithTx(tx pgx.Tx) commentMediaRepo
	Add(ctx context.Context, commentID uuid.UUID, mediaKey, mediaType string, position int32) (*entity.CommentMedia, error)
	GetByCommentsBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]*entity.CommentMedia, error)
}

// commentLikeRepo defines the repository operations needed by CommentLikeService.
// *repository.CommentLikeRepository satisfies this interface.
type commentLikeRepo interface {
	Like(ctx context.Context, userID, commentID uuid.UUID) error
	Unlike(ctx context.Context, userID, commentID uuid.UUID) error
	IsLiked(ctx context.Context, userID, commentID uuid.UUID) (bool, error)
	GetLikedCommentIDs(ctx context.Context, userID uuid.UUID, commentIDs []uuid.UUID) ([]uuid.UUID, error)
}

// userReader fetches author information for enriching posts.
// Implemented at the app layer to avoid cross-context imports.
type userReader interface {
	GetAuthorsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.Author, error)
}

// hashtagRepo defines the repository operations needed by PostService and HashtagService.
// *hashtagRepoTxable satisfies this interface in production.
type hashtagRepo interface {
	WithTx(tx pgx.Tx) hashtagRepo
	UpsertAndLink(ctx context.Context, postID uuid.UUID, names []string) error
	ReplaceForPost(ctx context.Context, postID uuid.UUID, names []string) error
	GetNamesByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]string, error)
	GetTrending(ctx context.Context, limit int32) ([]*entity.TrendingHashtag, error)
	GetPostsByHashtag(ctx context.Context, name string, cursorCreatedAt pgtype.Timestamptz, cursorPostID uuid.UUID, limit int32) ([]*entity.Post, error)
	SearchByPrefix(ctx context.Context, prefix string, limit int32) ([]string, error)
}

// mentionRepo defines the repository operations needed by PostService for mentions.
// *repository.MentionRepository satisfies this interface.
type mentionRepo interface {
	Insert(ctx context.Context, postID, userID uuid.UUID) error
	DeleteByPost(ctx context.Context, postID uuid.UUID) error
	GetByPost(ctx context.Context, postID uuid.UUID) ([]uuid.UUID, error)
	GetBatch(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
}

// commentMentionRepo defines the repository operations needed by CommentService for mentions.
// *repository.CommentMentionRepository satisfies this interface.
type commentMentionRepo interface {
	Insert(ctx context.Context, commentID, userID uuid.UUID) error
	GetBatch(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
}

// notificationEmitter emits mention notifications.
// Implemented at the app layer to avoid cross-context imports.
type notificationEmitter interface {
	EmitMention(ctx context.Context, actorID, recipientID, postID uuid.UUID) error
}

// TrendingInvalidator invalidates the trending posts cache.
// Defined here to avoid importing the feed package (would create a cycle).
type TrendingInvalidator interface {
	InvalidateTrending(ctx context.Context) error
}

// LikeNotificationEmitter emits like/unlike notifications for posts.
// Defined here to avoid importing the notification package (would create a cycle).
type LikeNotificationEmitter interface {
	EmitLike(ctx context.Context, actorID, recipientID, postID uuid.UUID) error
	DeleteNotification(ctx context.Context, actorID uuid.UUID, groupKey string) error
}

// CommentNotificationEmitter emits comment, reply, and mention notifications.
// Defined here to avoid importing the notification package (would create a cycle).
type CommentNotificationEmitter interface {
	EmitComment(ctx context.Context, actorID, recipientID, postID, commentID uuid.UUID) error
	EmitReply(ctx context.Context, actorID, recipientID, parentCommentID, replyID uuid.UUID) error
	EmitMention(ctx context.Context, actorID, recipientID, targetID uuid.UUID) error
}

// CommentLikeNotificationEmitter emits like/unlike notifications for comments.
// Defined here to avoid importing the notification package (would create a cycle).
type CommentLikeNotificationEmitter interface {
	EmitCommentLike(ctx context.Context, actorID, recipientID, commentID uuid.UUID) error
	DeleteNotification(ctx context.Context, actorID uuid.UUID, groupKey string) error
}

// hashtagCache defines caching operations for hashtag data.
// *cache.RedisHashtagCache and *cache.NopHashtagCache satisfy this interface.
type hashtagCache interface {
	GetTrendingHashtags(ctx context.Context) ([]*entity.TrendingHashtag, error)
	SetTrendingHashtags(ctx context.Context, tags []*entity.TrendingHashtag) error
	InvalidateTrendingHashtags(ctx context.Context) error
	GetHashtagPostsPage1(ctx context.Context, name string) (*postcache.HashtagPostsPage, error)
	SetHashtagPostsPage1(ctx context.Context, name string, page *postcache.HashtagPostsPage) error
	GetSearchResults(ctx context.Context, prefix string) ([]string, error)
	SetSearchResults(ctx context.Context, prefix string, names []string) error
}
