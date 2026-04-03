package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/repository"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

// CommentLikeService handles comment like/unlike business logic.
type CommentLikeService struct {
	commentLikeRepo commentLikeRepo
	commentRepo     commentRepo
	notifEmitter    CommentLikeNotificationEmitter // optional, nil = no-op
}

// NewCommentLikeService creates a new CommentLikeService.
func NewCommentLikeService(commentLikeRepo *repository.CommentLikeRepository, commentRepo *repository.CommentRepository) *CommentLikeService {
	return &CommentLikeService{commentLikeRepo: commentLikeRepo, commentRepo: &commentRepoTxable{commentRepo}}
}

// WithNotificationEmitter attaches a notification emitter. Called at wire-up time.
func (s *CommentLikeService) WithNotificationEmitter(e CommentLikeNotificationEmitter) {
	s.notifEmitter = e
}

// Toggle likes or unlikes a comment depending on current state. Returns true if now liked.
func (s *CommentLikeService) Toggle(ctx context.Context, userID, commentID uuid.UUID) (bool, error) {
	c, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return false, post.ErrCommentNotFound
		}
		return false, err
	}
	// if c.AuthorID == userID {
	// 	return false, post.ErrSelfCommentLike
	// }

	liked, err := s.commentLikeRepo.IsLiked(ctx, userID, commentID)
	if err != nil {
		logger.LogError(ctx, err, "failed to check comment like status", "user_id", userID, "comment_id", commentID)
		return false, errors.NewInternalError(err)
	}

	if liked {
		if err := s.commentLikeRepo.Unlike(ctx, userID, commentID); err != nil {
			logger.LogError(ctx, err, "failed to unlike comment", "user_id", userID, "comment_id", commentID)
			return false, errors.NewInternalError(err)
		}
		s.deleteCommentLikeNotification(ctx, userID, commentID)
		logger.Info(ctx, "comment unliked", "user_id", userID, "comment_id", commentID)
		return false, nil
	}

	if err := s.commentLikeRepo.Like(ctx, userID, commentID); err != nil {
		logger.LogError(ctx, err, "failed to like comment", "user_id", userID, "comment_id", commentID)
		return false, errors.NewInternalError(err)
	}
	s.emitCommentLikeNotification(ctx, userID, c.AuthorID, commentID)
	logger.Info(ctx, "comment liked", "user_id", userID, "comment_id", commentID)
	return true, nil
}

// --- notification helpers (fire-and-forget) ---

func (s *CommentLikeService) emitCommentLikeNotification(ctx context.Context, actorID, recipientID, commentID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.EmitCommentLike(ctx, actorID, recipientID, commentID); err != nil {
		logger.LogError(ctx, err, "failed to emit comment like notification", "actor", actorID, "comment", commentID)
	}
}

func (s *CommentLikeService) deleteCommentLikeNotification(ctx context.Context, actorID, commentID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.DeleteNotification(ctx, actorID, fmt.Sprintf("comment_like:%s", commentID)); err != nil {
		logger.LogError(ctx, err, "failed to delete comment like notification", "actor", actorID, "comment", commentID)
	}
}
