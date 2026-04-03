package feed

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

func TestScorer_Score(t *testing.T) {
	cfg := ScorerConfig{RelationshipBonus: 10, RecencyScale: 20, DecayExponent: 1.5}
	scorer := NewScorer(cfg)
	now := time.Now()

	makePost := func(likes int64, hoursAgo float64) *feedentity.Post {
		return &feedentity.Post{
			ID:        uuid.New(),
			LikeCount: likes,
			CreatedAt: now.Add(-time.Duration(hoursAgo * float64(time.Hour))),
		}
	}

	t.Run("relationship bonus applied when following", func(t *testing.T) {
		post := makePost(0, 0)
		withBonus := scorer.Score(post, true, now)
		withoutBonus := scorer.Score(post, false, now)
		if withBonus-withoutBonus != cfg.RelationshipBonus {
			t.Errorf("expected bonus=%.1f, got diff=%.4f", cfg.RelationshipBonus, withBonus-withoutBonus)
		}
	})

	t.Run("engagement increases score logarithmically", func(t *testing.T) {
		low := scorer.Score(makePost(10, 0), false, now)
		mid := scorer.Score(makePost(100, 0), false, now)
		high := scorer.Score(makePost(1000, 0), false, now)
		if !(low < mid && mid < high) {
			t.Error("expected score to increase with likes")
		}
		// Verify logarithmic compression: 1000 likes should not be 100x better than 10 likes
		if high/low > 10 {
			t.Errorf("expected logarithmic compression, got ratio %.2f", high/low)
		}
	})

	t.Run("recency decays over time", func(t *testing.T) {
		fresh := scorer.Score(makePost(0, 0), false, now)
		old := scorer.Score(makePost(0, 24), false, now)
		if fresh <= old {
			t.Errorf("expected fresh post (%.4f) to score higher than old post (%.4f)", fresh, old)
		}
	})

	t.Run("table examples", func(t *testing.T) {
		// A: 100 likes, 1h, following  → engagement≈46.2, recency=20/2^1.5≈7.1, bonus=10 → ~63.2
		scoreA := scorer.Score(makePost(100, 1), true, now)
		if math.Abs(scoreA-63.2) > 1.5 {
			t.Errorf("post A: expected ~63.2, got %.2f", scoreA)
		}

		// B: 500 likes, 2h, not following → engagement≈62.2, recency=20/3^1.5≈3.8, bonus=0 → ~66.0
		scoreB := scorer.Score(makePost(500, 2), false, now)
		if math.Abs(scoreB-66.0) > 1.5 {
			t.Errorf("post B: expected ~66.0, got %.2f", scoreB)
		}

		// C: 10 likes, 0.5h, following → engagement≈24.0, recency=20/1.5^1.5≈10.9, bonus=10 → ~44.9
		scoreC := scorer.Score(makePost(10, 0.5), true, now)
		if math.Abs(scoreC-44.9) > 1.5 {
			t.Errorf("post C: expected ~44.9, got %.2f", scoreC)
		}

		// Ordering: B > A > C
		// High engagement (500 likes) beats moderate engagement + relationship bonus.
		if !(scoreB > scoreA && scoreA > scoreC) {
			t.Errorf("expected B(%.2f) > A(%.2f) > C(%.2f)", scoreB, scoreA, scoreC)
		}
	})

	t.Run("future post treated as zero hours old", func(t *testing.T) {
		future := &feedentity.Post{
			ID:        uuid.New(),
			LikeCount: 0,
			CreatedAt: now.Add(1 * time.Hour),
		}
		score := scorer.Score(future, false, now)
		if score < 0 {
			t.Errorf("expected non-negative score, got %.4f", score)
		}
	})
}
