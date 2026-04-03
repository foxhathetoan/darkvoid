package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

var errSelfFollow = errors.New("SELF_FOLLOW", "cannot follow yourself", http.StatusBadRequest)

// FeedInvalidator is a narrow interface for cache invalidation.
// Defined here to avoid importing the feed package (would create a cycle).
type FeedInvalidator interface {
	InvalidateFollowingIDs(ctx context.Context, userID uuid.UUID) error
}

// FollowNotificationEmitter is a narrow interface for emitting follow notifications.
// Defined here to avoid importing the notification package (would create a cycle).
type FollowNotificationEmitter interface {
	EmitFollow(ctx context.Context, followerID, followeeID uuid.UUID) error
	DeleteNotification(ctx context.Context, actorID uuid.UUID, groupKey string) error
}

// FollowService handles follow/unfollow business logic.
type FollowService struct {
	followRepo      *repository.FollowRepository
	feedInvalidator FeedInvalidator           // optional, nil = no-op
	notifEmitter    FollowNotificationEmitter // optional, nil = no-op
}

func NewFollowService(followRepo *repository.FollowRepository) *FollowService {
	return &FollowService{followRepo: followRepo}
}

// WithFeedInvalidator attaches a cache invalidator. Called at wire-up time after
// the feed cache is initialized, avoiding circular init dependencies.
func (s *FollowService) WithFeedInvalidator(inv FeedInvalidator) {
	s.feedInvalidator = inv
}

// WithNotificationEmitter attaches a notification emitter. Called at wire-up time.
func (s *FollowService) WithNotificationEmitter(e FollowNotificationEmitter) {
	s.notifEmitter = e
}

func (s *FollowService) invalidateFollowingIDs(ctx context.Context, userID uuid.UUID) {
	if s.feedInvalidator == nil {
		return
	}
	if err := s.feedInvalidator.InvalidateFollowingIDs(ctx, userID); err != nil {
		logger.LogError(ctx, err, "failed to invalidate following IDs cache", "user_id", userID)
	}
}

func (s *FollowService) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return errSelfFollow
	}
	if err := s.followRepo.Follow(ctx, followerID, followeeID); err != nil {
		logger.LogError(ctx, err, "failed to follow", "follower", followerID, "followee", followeeID)
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "followed", "follower", followerID, "followee", followeeID)
	s.invalidateFollowingIDs(ctx, followerID)
	s.emitFollowNotification(ctx, followerID, followeeID)
	return nil
}

func (s *FollowService) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return errSelfFollow
	}
	if err := s.followRepo.Unfollow(ctx, followerID, followeeID); err != nil {
		logger.LogError(ctx, err, "failed to unfollow", "follower", followerID, "followee", followeeID)
		return errors.NewInternalError(err)
	}
	logger.Info(ctx, "unfollowed", "follower", followerID, "followee", followeeID)
	s.invalidateFollowingIDs(ctx, followerID)
	s.deleteFollowNotification(ctx, followerID, followeeID)
	return nil
}

// --- notification helpers (fire-and-forget) ---

func (s *FollowService) emitFollowNotification(ctx context.Context, followerID, followeeID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.EmitFollow(ctx, followerID, followeeID); err != nil {
		logger.LogError(ctx, err, "failed to emit follow notification", "follower", followerID, "followee", followeeID)
	}
}

func (s *FollowService) deleteFollowNotification(ctx context.Context, followerID, followeeID uuid.UUID) {
	if s.notifEmitter == nil {
		return
	}
	if err := s.notifEmitter.DeleteNotification(ctx, followerID, fmt.Sprintf("follow:%s", followeeID)); err != nil {
		logger.LogError(ctx, err, "failed to delete follow notification", "follower", followerID, "followee", followeeID)
	}
}

func (s *FollowService) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	ok, err := s.followRepo.IsFollowing(ctx, followerID, followeeID)
	if err != nil {
		logger.LogError(ctx, err, "failed to check following", "follower", followerID, "followee", followeeID)
		return false, errors.NewInternalError(err)
	}
	return ok, nil
}

func (s *FollowService) GetFollowers(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
	req.Validate()
	follows, err := s.followRepo.GetFollowers(ctx, targetID, req.Limit, req.Offset)
	if err != nil {
		logger.LogError(ctx, err, "failed to get followers", "user_id", targetID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}
	total, err := s.followRepo.CountFollowers(ctx, targetID)
	if err != nil {
		logger.LogError(ctx, err, "failed to count followers", "user_id", targetID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}
	return follows, pagination.NewPaginationResponse(total, req.Limit, req.Offset), nil
}

func (s *FollowService) GetFollowingIDs(ctx context.Context, targetID uuid.UUID) ([]uuid.UUID, error) {
	follows, err := s.followRepo.GetFollowing(ctx, targetID, 5000, 0)
	if err != nil {
		logger.LogError(ctx, err, "failed to get following IDs", "user_id", targetID)
		return nil, errors.NewInternalError(err)
	}
	ids := make([]uuid.UUID, len(follows))
	for i, f := range follows {
		ids[i] = f.FolloweeID
	}
	return ids, nil
}

// GetFollowerIDs returns the IDs of all users who follow targetID.
func (s *FollowService) GetFollowerIDs(ctx context.Context, targetID uuid.UUID) ([]uuid.UUID, error) {
	follows, err := s.followRepo.GetFollowers(ctx, targetID, 5000, 0)
	if err != nil {
		logger.LogError(ctx, err, "failed to get follower IDs", "user_id", targetID)
		return nil, errors.NewInternalError(err)
	}
	ids := make([]uuid.UUID, len(follows))
	for i, f := range follows {
		ids[i] = f.FollowerID
	}
	return ids, nil
}

func (s *FollowService) GetFollowing(ctx context.Context, targetID uuid.UUID, req pagination.PaginationRequest) ([]*entity.Follow, pagination.PaginationResponse, error) {
	req.Validate()
	follows, err := s.followRepo.GetFollowing(ctx, targetID, req.Limit, req.Offset)
	if err != nil {
		logger.LogError(ctx, err, "failed to get following", "user_id", targetID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}
	total, err := s.followRepo.CountFollowing(ctx, targetID)
	if err != nil {
		logger.LogError(ctx, err, "failed to count following", "user_id", targetID)
		return nil, pagination.PaginationResponse{}, errors.NewInternalError(err)
	}
	return follows, pagination.NewPaginationResponse(total, req.Limit, req.Offset), nil
}
