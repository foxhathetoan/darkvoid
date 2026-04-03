package handler

import (
	"context"

	"github.com/google/uuid"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/pagination"
)

// postService defines the methods used by PostHandler.
// *service.PostService satisfies this interface.
type postService interface {
	CreatePost(ctx context.Context, authorID uuid.UUID, content string, visibility entity.Visibility, mediaKeys []string, mentionUserIDs []uuid.UUID, tags []string) (*entity.Post, error)
	GetPost(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID) (*entity.Post, error)
	UpdatePost(ctx context.Context, postID, userID uuid.UUID, content string, visibility entity.Visibility, mentionUserIDs []uuid.UUID, tags []string) (*entity.Post, error)
	DeletePost(ctx context.Context, postID, userID uuid.UUID) error
	GetUserPosts(ctx context.Context, authorID uuid.UUID, viewerID *uuid.UUID, cursor *post.UserPostCursor, visibility string, limit int32) ([]*entity.Post, *post.UserPostCursor, error)
}

// commentService defines the methods used by CommentHandler.
// *service.CommentService satisfies this interface.
type commentService interface {
	CreateComment(ctx context.Context, postID, authorID uuid.UUID, parentID *uuid.UUID, content string, mediaKeys []string, mentionUserIDs []uuid.UUID) (*entity.Comment, error)
	GetComments(ctx context.Context, postID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error)
	GetReplies(ctx context.Context, commentID uuid.UUID, viewerID *uuid.UUID, req pagination.PaginationRequest) ([]*entity.Comment, pagination.PaginationResponse, error)
	DeleteComment(ctx context.Context, commentID, userID uuid.UUID) error
}

// likeService defines the methods used by LikeHandler.
// *service.LikeService satisfies this interface.
type likeService interface {
	Toggle(ctx context.Context, userID, postID uuid.UUID) (bool, error)
}

// commentLikeService defines the methods used by CommentLikeHandler.
// *service.CommentLikeService satisfies this interface.
type commentLikeService interface {
	Toggle(ctx context.Context, userID, commentID uuid.UUID) (bool, error)
}

// hashtagSvc defines the methods used by HashtagHandler.
// *service.HashtagService satisfies this interface.
type hashtagSvc interface {
	GetTrending(ctx context.Context) ([]*entity.TrendingHashtag, error)
	SearchHashtags(ctx context.Context, prefix string) ([]string, error)
	GetPostsByHashtag(ctx context.Context, name string, viewerID *uuid.UUID, cursor *post.UserPostCursor, limit int32) ([]*entity.Post, *post.UserPostCursor, error)
}
