package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/feed"
	feedcache "github.com/jarviisha/darkvoid/internal/feature/feed/cache"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
	"github.com/jarviisha/darkvoid/pkg/errors"
	"github.com/jarviisha/darkvoid/pkg/logger"
)

const scoreEpsilon = 1e-9

const (
	pageSize           = 20
	fetchMultiplier    = 3
	trendingFetchLimit = 100
)

// FeedService handles feed business logic.
type FeedService struct {
	postReader   feed.PostReader
	followReader feed.FollowReader
	likeReader   feed.LikeReader
	ranker       feed.Ranker
	cache        feedcache.FeedCache
}

// NewFeedService creates a new FeedService.
func NewFeedService(postReader feed.PostReader, followReader feed.FollowReader, likeReader feed.LikeReader, ranker feed.Ranker, cache feedcache.FeedCache) *FeedService {
	return &FeedService{
		postReader:   postReader,
		followReader: followReader,
		likeReader:   likeReader,
		ranker:       ranker,
		cache:        cache,
	}
}

// GetFeed returns the cursor-paginated feed for userID.
// Page 1 (cursor == nil): mix of following + trending posts, scored and sorted.
// Page 2+ (cursor != nil): pure following posts in DB order (newest first).
// Cache strategy:
//   - HIT  following:ids:{userID}  → skip follow DB query
//   - MISS following:ids:{userID}  → query DB, store result in cache
//   - HIT  trending:posts          → skip trending DB query (page 1 only)
//   - MISS trending:posts          → query DB, store result in cache
func (s *FeedService) GetFeed(ctx context.Context, userID uuid.UUID, cursor *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error) {
	isFirstPage := cursor == nil

	// Already in discover mode → continue discover pagination directly.
	if cursor != nil && cursor.Mode == feed.ModeDiscover {
		return s.discoverFallback(ctx, userID, cursor)
	}

	// 1. Resolve following IDs (cache-aware) + include user's own posts.
	// Copy before appending to avoid mutating the cached slice's backing array.
	cachedIDs, err := s.getFollowingIDs(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	authorIDs := make([]uuid.UUID, len(cachedIDs)+1)
	copy(authorIDs, cachedIDs)
	authorIDs[len(cachedIDs)] = userID

	followingSet := make(map[uuid.UUID]bool, len(authorIDs))
	for _, id := range authorIDs {
		followingSet[id] = true
	}

	// 2. Fetch following posts from DB with cursor (fetch buffer for dedup/merge)
	followingPosts, err := s.postReader.GetFollowingPostsWithCursor(ctx, authorIDs, cursor, pageSize*fetchMultiplier)
	if err != nil {
		logger.LogError(ctx, err, "failed to get following posts", "user_id", userID)
		return nil, nil, errors.NewInternalError(err)
	}

	// 3. No following content → fall back to discover (infinite scroll through public posts).
	// Applies to both new users (page 1) and users who've exhausted their following feed (page N).
	// Pass nil cursor so discover starts from the newest post, not from the following cursor position.
	if len(followingPosts) == 0 {
		return s.discoverFallback(ctx, userID, nil)
	}

	if isFirstPage {
		// Page 1: merge following + trending, score all, sort, return top pageSize.
		now := time.Now()
		followingStrSet := make(map[string]bool, len(followingSet))
		for id := range followingSet {
			followingStrSet[id.String()] = true
		}

		trendingPosts, err := s.getTrending(ctx)
		if err != nil {
			// Non-fatal: serve following-only feed on trending failure.
			logger.LogError(ctx, err, "failed to get trending posts, skipping", "user_id", userID)
		}

		// Dedup and collect all posts + sources for batch ranking.
		seen := make(map[uuid.UUID]bool)
		type postWithSource struct {
			post   *feedentity.Post
			source feedentity.Source
		}
		allPosts := make([]postWithSource, 0, len(followingPosts)+len(trendingPosts))

		for _, p := range followingPosts {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			allPosts = append(allPosts, postWithSource{post: p, source: feedentity.SourceFollowing})
		}
		for _, p := range trendingPosts {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			allPosts = append(allPosts, postWithSource{post: p, source: feedentity.SourceTrending})
		}

		// Batch rank all posts.
		postsToRank := make([]*feedentity.Post, len(allPosts))
		for i, ps := range allPosts {
			postsToRank[i] = ps.post
		}
		scores, err := s.ranker.RankPosts(ctx, postsToRank, followingStrSet, now)
		if err != nil {
			logger.LogError(ctx, err, "ranker failed, falling back to chronological order", "user_id", userID)
			// Fallback: use 0 scores so sort falls through to created_at ordering.
			scores = make(map[string]float64)
		}

		items := make([]*feedentity.FeedItem, len(allPosts))
		for i, ps := range allPosts {
			items[i] = &feedentity.FeedItem{
				Post:   ps.post,
				Score:  scores[ps.post.ID.String()],
				Source: ps.source,
			}
		}

		// Sort descending by (score, created_at, post_id).
		sort.Slice(items, func(i, j int) bool {
			if math.Abs(items[i].Score-items[j].Score) > scoreEpsilon {
				return items[i].Score > items[j].Score
			}
			if !items[i].Post.CreatedAt.Equal(items[j].Post.CreatedAt) {
				return items[i].Post.CreatedAt.After(items[j].Post.CreatedAt)
			}
			return items[i].Post.ID.String() > items[j].Post.ID.String()
		})

		page := items
		if len(page) > pageSize {
			page = page[:pageSize]
		}

		s.enrichIsLiked(ctx, userID, page)

		// Cursor = position of the last following post shown on page 1.
		// Page 2 will continue the following timeline from there.
		var nextCursor *feed.FollowingCursor
		for i := len(page) - 1; i >= 0; i-- {
			if page[i].Source == feedentity.SourceFollowing {
				nextCursor = &feed.FollowingCursor{
					Mode:      feed.ModeFollowing,
					CreatedAt: page[i].Post.CreatedAt,
					PostID:    page[i].Post.ID.String(),
				}
				break
			}
		}
		// If no following post made it onto page 1 (all trending won the ranking),
		// use a far-future sentinel so page 2 fetches from the top of the following timeline.
		if nextCursor == nil && len(followingPosts) > 0 {
			nextCursor = &feed.FollowingCursor{
				Mode:      feed.ModeFollowing,
				CreatedAt: feed.MaxDiscoverTime,
				PostID:    uuid.Max.String(),
			}
		}

		return page, nextCursor, nil
	}

	// Page 2+: pure following posts in DB order, no trending injection or scoring.
	items := make([]*feedentity.FeedItem, 0, pageSize)
	for _, p := range followingPosts {
		items = append(items, &feedentity.FeedItem{
			Post:   p,
			Source: feedentity.SourceFollowing,
		})
	}

	var nextCursor *feed.FollowingCursor
	if len(items) > pageSize {
		last := items[pageSize-1]
		nextCursor = &feed.FollowingCursor{
			Mode:      feed.ModeFollowing,
			CreatedAt: last.Post.CreatedAt,
			PostID:    last.Post.ID.String(),
		}
		items = items[:pageSize]
	}

	s.enrichIsLiked(ctx, userID, items)
	return items, nextCursor, nil
}

// getFollowingIDs returns following IDs from cache, falling back to followReader on miss.
func (s *FeedService) getFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if ids, err := s.cache.GetFollowingIDs(ctx, userID); err != nil {
		logger.LogError(ctx, err, "following IDs cache read error, falling through to DB", "user_id", userID)
	} else if ids != nil {
		return ids, nil
	}

	ids, err := s.followReader.GetFollowingIDs(ctx, userID)
	if err != nil {
		logger.LogError(ctx, err, "failed to resolve following IDs", "user_id", userID)
		return nil, errors.NewInternalError(err)
	}

	if err := s.cache.SetFollowingIDs(ctx, userID, ids); err != nil {
		logger.LogError(ctx, err, "following IDs cache write error", "user_id", userID)
	}
	return ids, nil
}

// discoverFallback paginates through the public discover feed when the user has no following content.
// Used for new users and users who've scrolled past all their following posts.
// FollowingCursor and DiscoverCursor share the same (CreatedAt, PostID) fields, so the cursor
// transitions seamlessly between the two feed modes.
func (s *FeedService) discoverFallback(ctx context.Context, userID uuid.UUID, cursor *feed.FollowingCursor) ([]*feedentity.FeedItem, *feed.FollowingCursor, error) {
	var discoverCursor *feed.DiscoverCursor
	if cursor != nil && cursor.Mode == feed.ModeDiscover {
		discoverCursor = &feed.DiscoverCursor{
			CreatedAt: cursor.CreatedAt,
			PostID:    cursor.PostID,
		}
	}
	// When cursor is nil or Mode is ModeFollowing, discoverCursor stays nil
	// → discover starts from the newest post.

	// Fetch pageSize+1 to detect whether there is a next page.
	posts, err := s.postReader.GetDiscoverWithCursor(ctx, discoverCursor, pageSize+1, nil)
	if err != nil {
		logger.LogError(ctx, err, "failed to get discover fallback", "user_id", userID)
		return nil, nil, errors.NewInternalError(err)
	}

	var nextCursor *feed.FollowingCursor
	if len(posts) > pageSize {
		last := posts[pageSize-1]
		nextCursor = &feed.FollowingCursor{
			Mode:      feed.ModeDiscover,
			CreatedAt: last.CreatedAt,
			PostID:    last.ID.String(),
		}
		posts = posts[:pageSize]
	}

	now := time.Now()
	emptyFollowing := map[string]bool{} // discover = no relationship bonus
	scores, rankErr := s.ranker.RankPosts(ctx, posts, emptyFollowing, now)
	if rankErr != nil {
		logger.LogError(ctx, rankErr, "ranker failed in discover fallback", "user_id", userID)
		scores = make(map[string]float64)
	}
	items := make([]*feedentity.FeedItem, 0, len(posts))
	for _, p := range posts {
		items = append(items, &feedentity.FeedItem{
			Post:   p,
			Score:  scores[p.ID.String()],
			Source: feedentity.SourceDiscover,
		})
	}
	s.enrichIsLiked(ctx, userID, items)
	return items, nextCursor, nil
}

// enrichIsLiked batch-fetches like status for the viewer and sets Post.IsLiked.
// Best-effort: on error, items are returned as-is (is_liked stays false).
func (s *FeedService) enrichIsLiked(ctx context.Context, userID uuid.UUID, items []*feedentity.FeedItem) {
	if len(items) == 0 {
		return
	}
	postIDs := make([]uuid.UUID, len(items))
	for i, item := range items {
		postIDs[i] = item.Post.ID
	}
	likedIDs, err := s.likeReader.GetLikedPostIDs(ctx, userID, postIDs)
	if err != nil {
		logger.LogError(ctx, err, "failed to enrich is_liked for feed", "user_id", userID)
		return
	}
	likedSet := make(map[uuid.UUID]bool, len(likedIDs))
	for _, id := range likedIDs {
		likedSet[id] = true
	}
	for _, item := range items {
		item.Post.IsLiked = likedSet[item.Post.ID]
	}
}

// getTrending returns trending posts from cache, falling back to DB on miss.
func (s *FeedService) getTrending(ctx context.Context) ([]*feedentity.Post, error) {
	if cached, err := s.cache.GetTrending(ctx); err != nil {
		logger.LogError(ctx, err, "trending cache read error, falling through to DB")
	} else if cached != nil {
		return cached, nil
	}

	posts, err := s.postReader.GetTrendingPosts(ctx, trendingFetchLimit)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetTrending(ctx, posts); err != nil {
		logger.LogError(ctx, err, "trending cache write error")
	}

	return posts, nil
}

// GetDiscover returns the cursor-paginated public discovery feed.
// viewerID is optional — when provided, is_liked is populated.
func (s *FeedService) GetDiscover(ctx context.Context, viewerID *uuid.UUID, cursor *feed.DiscoverCursor, limit int32) ([]*feedentity.Post, *feed.DiscoverCursor, error) {
	const defaultLimit = 20
	if limit <= 0 {
		limit = defaultLimit
	}
	// Fetch one extra to detect if there's a next page
	posts, err := s.postReader.GetDiscoverWithCursor(ctx, cursor, limit+1, viewerID)
	if err != nil {
		logger.LogError(ctx, err, "failed to get discover feed")
		return nil, nil, err
	}

	var nextCursor *feed.DiscoverCursor
	if len(posts) > int(limit) {
		last := posts[limit-1]
		nextCursor = &feed.DiscoverCursor{
			CreatedAt: last.CreatedAt,
			PostID:    last.ID.String(),
		}
		posts = posts[:limit]
	}

	return posts, nextCursor, nil
}
