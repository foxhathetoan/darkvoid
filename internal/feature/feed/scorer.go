package feed

import (
	"math"
	"time"

	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
)

// ScorerConfig controls feed ranking behaviour.
type ScorerConfig struct {
	RelationshipBonus float64 // bonus added when post author is followed (default 10)
	RecencyScale      float64 // multiplier for recency signal (default 20); max recency = RecencyScale at t=0
	DecayExponent     float64 // power applied to recency decay (default 1.5)
}

// DefaultScorerConfig returns the default configuration.
func DefaultScorerConfig() ScorerConfig {
	return ScorerConfig{
		RelationshipBonus: 10,
		RecencyScale:      20,
		DecayExponent:     1.5,
	}
}

// Scorer computes a ranking score for a post.
// It is pure and stateless — safe to use concurrently.
type Scorer struct {
	cfg ScorerConfig
}

// NewScorer creates a Scorer with the given config.
func NewScorer(cfg ScorerConfig) *Scorer {
	return &Scorer{cfg: cfg}
}

// Score returns the ranking score for a post.
//
//	score = engagement_score + recency_score + relationship_bonus
//	engagement_score   = log(1 + like_count) × 10
//	recency_score      = RecencyScale / (1 + hours)^decay_exponent
//	relationship_bonus = cfg.RelationshipBonus (if isFollowing), else 0
//
// All three components share a comparable scale (~0–20 each for typical posts),
// so no single signal dominates the others.
func (s *Scorer) Score(post *feedentity.Post, isFollowing bool, now time.Time) float64 {
	hours := now.Sub(post.CreatedAt).Hours()
	if hours < 0 {
		hours = 0
	}

	engagement := math.Log1p(float64(post.LikeCount)) * 10
	recency := s.cfg.RecencyScale / math.Pow(1+hours, s.cfg.DecayExponent)

	bonus := 0.0
	if isFollowing {
		bonus = s.cfg.RelationshipBonus
	}

	return engagement + recency + bonus
}
