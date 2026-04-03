package feed

import (
	"context"
	"time"

	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
	"github.com/jarviisha/darkvoid/pkg/logger"
	dikeclient "github.com/jarviisha/dike/pkg/client"
	rankingv1 "github.com/jarviisha/dike/proto/ranking/v1"
)

// DikeRanker implements Ranker by delegating scoring to the Dike ranking service via gRPC.
type DikeRanker struct {
	client  *dikeclient.Client
	modelID string // optional: override default model per call
}

// NewDikeRanker creates a DikeRanker wrapping the given Dike client.
// modelID can be empty to use the project's default model.
func NewDikeRanker(client *dikeclient.Client, modelID string) *DikeRanker {
	return &DikeRanker{client: client, modelID: modelID}
}

// RankPosts converts posts to Dike items, calls the ranking service, and returns scores keyed by post ID.
func (r *DikeRanker) RankPosts(ctx context.Context, posts []*feedentity.Post, followingSet map[string]bool, now time.Time) (map[string]float64, error) {
	if len(posts) == 0 {
		return map[string]float64{}, nil
	}

	items := make([]*rankingv1.Item, len(posts))
	for i, p := range posts {
		isFollowing := 0.0
		if followingSet[p.AuthorID.String()] {
			isFollowing = 1.0
		}

		hours := now.Sub(p.CreatedAt).Hours()
		if hours < 0 {
			hours = 0
		}

		items[i] = &rankingv1.Item{
			ItemId: p.ID.String(),
			Features: map[string]float64{
				"engagement_count": float64(p.LikeCount),
				"comment_count":    float64(p.CommentCount),
				"hours_age":        hours,
				"is_following":     isFollowing,
			},
		}
	}

	var opts []dikeclient.ScoreOption
	if r.modelID != "" {
		opts = append(opts, dikeclient.WithScoreModel(r.modelID))
	}

	result, err := r.client.Score(ctx, items, opts...)
	if err != nil {
		return nil, err
	}

	if len(result.Warnings) > 0 {
		logger.Warn(ctx, "dike scoring warnings", "warnings", result.Warnings)
	}

	scores := make(map[string]float64, len(result.Items))
	for _, item := range result.Items {
		scores[item.ItemId] = item.Score
	}
	return scores, nil
}

// Close closes the underlying Dike gRPC connection.
func (r *DikeRanker) Close() error {
	return r.client.Close()
}
