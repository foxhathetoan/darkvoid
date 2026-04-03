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

// LikeService handles like/unlike business logic
type LikeService struct {
	likeRepo            likeRepo
	postRepo            postRepo
	trendingInvalidator TrendingInvalidator     // optional, nil = no-op
	notifEmitter        LikeNotificationEmitter // optional, nil = no-op
}

// NewLikeService creates a new LikeService
func NewLikeService(likeRepo *repository.LikeRepository, postRepo *repository.PostRepository) *LikeService {
	return &LikeService{likeRepo: likeRepo, postRepo: &postRepoTxable{postRepo}}
}

// WithTrendingInvalidator attaches a trending cache invalidator. Called at wire-up time.
func (s *LikeService) WithTrendingInvalidator(inv TrendingInvalidator) {
	s.trendingInvalidator = inv
}

// WithNotificationEmitter attaches a notification emitter. Called at wire-up time.
func (s *LikeService) WithNotificationEmitter(e LikeNotificationEmitter) {
	s.notifEmitter = e
}

func (s *LikeService) invalidateTrending(ctx context.Context) {
	if s.trendingInvalidator == nil {
		return
	}
	if err := s.trendingInvalidator.InvalidateTrending(ctx); err != nil {
		logger.LogError(ctx, err, "failed to invalidate trending cache")
	}
}

// Like adds a like from userID to postID
func (s *LikeService) Like(ctx context.Context, userID, postID uuid.UUID) error {
	p, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return post.ErrPostNotFound
		}
		return err
	}
	if p.AuthorID == userID {
		return post.ErrSelfLike
	}

	if err := s.likeRepo.Like(ctx, userID, postID); err != nil {
		logger.LogError(ctx, err, "failed to like post", "user_id", userID, "post_id", postID)
		return errors.NewInternalError(err)
	}
	s.invalidateTrending(ctx)
	s.emitLikeNotification(ctx, userID, p.AuthorID, postID)
	logger.Info(ctx, "post liked", "user_id", userID, "post_id", postID)
	return nil
}

// Unlike removes a like from userID to postID
func (s *LikeService) Unlike(ctx context.Context, userID, postID uuid.UUID) error {
	if _, err := s.postRepo.GetByID(ctx, postID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return post.ErrPostNotFound
		}
		return err
	}

	if err := s.likeRepo.Unlike(ctx, userID, postID); err != nil {
		logger.LogError(ctx, err, "failed to unlike post", "user_id", userID, "post_id", postID)
		return errors.NewInternalError(err)
	}
	s.invalidateTrending(ctx)
	s.deleteLikeNotification(ctx, userID, postID)
	logger.Info(ctx, "post unliked", "user_id", userID, "post_id", postID)
	return nil
}

// Toggle likes or unlikes a post depending on current state. Returns true if now liked.
func (s *LikeService) Toggle(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	p, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return false, post.ErrPostNotFound
		}
		return false, err
	}
	// if p.AuthorID == userID {
	// 	return false, post.ErrSelfLike
	// }

	// IsLiked + Like/Unlike is not atomic, but safe: LikePost uses ON CONFLICT DO NOTHING
	// and UnlikePost is an idempotent DELETE, so concurrent requests cannot corrupt state.
	liked, err := s.likeRepo.IsLiked(ctx, userID, postID)
	if err != nil {
		logger.LogError(ctx, err, "failed to check like status", "user_id", userID, "post_id", postID)
		return false, errors.NewInternalError(err)
	}

	if liked {
		if err := s.likeRepo.Unlike(ctx, userID, postID); err != nil {
			logger.LogError(ctx, err, "failed to unlike post", "user_id", userID, "post_id", postID)
			return false, errors.NewInternalError(err)
		}
		s.invalidateTrending(ctx)
		s.deleteLikeNotification(ctx, userID, postID)
		logger.Info(ctx, "post unliked", "user_id", userID, "post_id", postID)
		return false, nil
	}

	if err := s.likeRepo.Like(ctx, userID, postID); err != nil {
		logger.LogError(ctx, err, "failed to like post", "user_id", userID, "post_id", postID)
		return false, errors.NewInternalError(err)
	}
	s.invalidateTrending(ctx)
	s.emitLikeNotification(ctx, userID, p.AuthorID, postID)
	logger.Info(ctx, "post liked", "user_id", userID, "post_id", postID)
	return true, nil
}

// --- notification helpers (fire-and-forget) ---

func (s *LikeService) emitLikeNotification(ctx context.Context, actorID, recipientID, postID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.EmitLike(ctx, actorID, recipientID, postID); err != nil {
		logger.LogError(ctx, err, "failed to emit like notification", "actor", actorID, "post", postID)
	}
}

func (s *LikeService) deleteLikeNotification(ctx context.Context, actorID, postID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.DeleteNotification(ctx, actorID, fmt.Sprintf("like:%s", postID)); err != nil {
		logger.LogError(ctx, err, "failed to delete like notification", "actor", actorID, "post", postID)
	}
}
