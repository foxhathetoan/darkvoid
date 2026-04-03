package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	feedentity "github.com/jarviisha/darkvoid/internal/feature/feed/entity"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
	"github.com/redis/go-redis/v9"
)

const (
	trendingKey = "trending:posts"
)

func followingIDsKey(userID uuid.UUID) string {
	return fmt.Sprintf("following:ids:%s", userID)
}

// RedisFeedCache implements FeedCache using Redis.
type RedisFeedCache struct {
	client *pkgredis.Client
}

func NewRedisFeedCache(client *pkgredis.Client) *RedisFeedCache {
	return &RedisFeedCache{client: client}
}

// -- Following IDs cache --

func (c *RedisFeedCache) GetFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	raw, err := c.client.Get(ctx, followingIDsKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get following IDs: %w", err)
	}
	var ids []uuid.UUID
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("unmarshal following IDs: %w", err)
	}
	return ids, nil
}

func (c *RedisFeedCache) SetFollowingIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	raw, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal following IDs: %w", err)
	}
	if err := c.client.Set(ctx, followingIDsKey(userID), raw, FollowingIDsTTL).Err(); err != nil {
		return fmt.Errorf("redis set following IDs: %w", err)
	}
	return nil
}

func (c *RedisFeedCache) InvalidateFollowingIDs(ctx context.Context, userID uuid.UUID) error {
	if err := c.client.Del(ctx, followingIDsKey(userID)).Err(); err != nil {
		return fmt.Errorf("redis del following IDs: %w", err)
	}
	return nil
}

// -- Trending cache --

func (c *RedisFeedCache) GetTrending(ctx context.Context) ([]*feedentity.Post, error) {
	raw, err := c.client.Get(ctx, trendingKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get trending: %w", err)
	}

	var posts []*feedentity.Post
	if err := json.Unmarshal(raw, &posts); err != nil {
		return nil, fmt.Errorf("unmarshal trending: %w", err)
	}
	return posts, nil
}

func (c *RedisFeedCache) SetTrending(ctx context.Context, posts []*feedentity.Post) error {
	raw, err := json.Marshal(posts)
	if err != nil {
		return fmt.Errorf("marshal trending: %w", err)
	}
	if err := c.client.Set(ctx, trendingKey, raw, TrendingTTL).Err(); err != nil {
		return fmt.Errorf("redis set trending: %w", err)
	}
	return nil
}

func (c *RedisFeedCache) InvalidateTrending(ctx context.Context) error {
	if err := c.client.Del(ctx, trendingKey).Err(); err != nil {
		return fmt.Errorf("redis del trending: %w", err)
	}
	return nil
}
