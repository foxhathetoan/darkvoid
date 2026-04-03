package feed

import (
	"context"
	"math"
	"time"

	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

// RankedItem holds a post ID and its computed score.
type RankedItem struct {
	PostID string
	Score  float64
}

// Ranker scores a batch of posts and returns ranked results.
// Implementations include the local formula-based Scorer and the remote Dike gRPC client.
type Ranker interface {
	// RankPosts scores the given posts and returns results keyed by post ID.
	// followingSet indicates which author IDs the viewer follows (for relationship bonus).
	RankPosts(ctx context.Context, posts []*feedentity.Post, followingSet map[string]bool, now time.Time) (map[string]float64, error)
}

// LocalRanker wraps the existing Scorer to implement the Ranker interface.
type LocalRanker struct {
	cfg ScorerConfig
}

// NewLocalRanker creates a LocalRanker with the given config.
func NewLocalRanker(cfg ScorerConfig) *LocalRanker {
	return &LocalRanker{cfg: cfg}
}

// RankPosts scores posts using the local formula.
func (r *LocalRanker) RankPosts(_ context.Context, posts []*feedentity.Post, followingSet map[string]bool, now time.Time) (map[string]float64, error) {
	scores := make(map[string]float64, len(posts))
	for _, p := range posts {
		hours := now.Sub(p.CreatedAt).Hours()
		if hours < 0 {
			hours = 0
		}

		engagement := math.Log1p(float64(p.LikeCount)) * 10
		recency := r.cfg.RecencyScale / math.Pow(1+hours, r.cfg.DecayExponent)

		bonus := 0.0
		if followingSet[p.AuthorID.String()] {
			bonus = r.cfg.RelationshipBonus
		}

		scores[p.ID.String()] = engagement + recency + bonus
	}
	return scores, nil
}
