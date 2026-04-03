package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
	"github.com/redis/go-redis/v9"
)

const trendingHashtagsKey = "trending:hashtags"

func hashtagPostsPage1Key(name string) string {
	return "hashtag:posts:page1:" + name
}

func hashtagSearchKey(prefix string) string {
	return "hashtag:search:" + prefix
}

// RedisHashtagCache implements HashtagCache using Redis.
type RedisHashtagCache struct {
	client *pkgredis.Client
}

// NewRedisHashtagCache creates a new RedisHashtagCache.
func NewRedisHashtagCache(client *pkgredis.Client) *RedisHashtagCache {
	return &RedisHashtagCache{client: client}
}

func (c *RedisHashtagCache) GetTrendingHashtags(ctx context.Context) ([]*entity.TrendingHashtag, error) {
	raw, err := c.client.Get(ctx, trendingHashtagsKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get trending hashtags: %w", err)
	}
	var tags []*entity.TrendingHashtag
	if err := json.Unmarshal(raw, &tags); err != nil {
		return nil, fmt.Errorf("unmarshal trending hashtags: %w", err)
	}
	return tags, nil
}

func (c *RedisHashtagCache) SetTrendingHashtags(ctx context.Context, tags []*entity.TrendingHashtag) error {
	raw, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("marshal trending hashtags: %w", err)
	}
	if err := c.client.Set(ctx, trendingHashtagsKey, raw, TrendingHashtagsTTL).Err(); err != nil {
		return fmt.Errorf("redis set trending hashtags: %w", err)
	}
	return nil
}

func (c *RedisHashtagCache) InvalidateTrendingHashtags(ctx context.Context) error {
	if err := c.client.Del(ctx, trendingHashtagsKey).Err(); err != nil {
		return fmt.Errorf("redis del trending hashtags: %w", err)
	}
	return nil
}

func (c *RedisHashtagCache) GetHashtagPostsPage1(ctx context.Context, name string) (*HashtagPostsPage, error) {
	raw, err := c.client.Get(ctx, hashtagPostsPage1Key(name)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil //nolint:nilnil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get hashtag posts page1 %q: %w", name, err)
	}
	var page HashtagPostsPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, fmt.Errorf("unmarshal hashtag posts page1 %q: %w", name, err)
	}
	return &page, nil
}

func (c *RedisHashtagCache) SetHashtagPostsPage1(ctx context.Context, name string, page *HashtagPostsPage) error {
	raw, err := json.Marshal(page)
	if err != nil {
		return fmt.Errorf("marshal hashtag posts page1 %q: %w", name, err)
	}
	if err := c.client.Set(ctx, hashtagPostsPage1Key(name), raw, HashtagPostsPage1TTL).Err(); err != nil {
		return fmt.Errorf("redis set hashtag posts page1 %q: %w", name, err)
	}
	return nil
}

func (c *RedisHashtagCache) GetSearchResults(ctx context.Context, prefix string) ([]string, error) {
	raw, err := c.client.Get(ctx, hashtagSearchKey(prefix)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get hashtag search %q: %w", prefix, err)
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, fmt.Errorf("unmarshal hashtag search %q: %w", prefix, err)
	}
	return names, nil
}

func (c *RedisHashtagCache) SetSearchResults(ctx context.Context, prefix string, names []string) error {
	raw, err := json.Marshal(names)
	if err != nil {
		return fmt.Errorf("marshal hashtag search %q: %w", prefix, err)
	}
	if err := c.client.Set(ctx, hashtagSearchKey(prefix), raw, HashtagSearchTTL).Err(); err != nil {
		return fmt.Errorf("redis set hashtag search %q: %w", prefix, err)
	}
	return nil
}
