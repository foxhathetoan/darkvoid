package dto

import (
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// TrendingHashtagResponse represents one trending hashtag with its usage count.
type TrendingHashtagResponse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// TrendingHashtagListResponse is the response for the trending hashtags endpoint.
type TrendingHashtagListResponse struct {
	Data []TrendingHashtagResponse `json:"data"`
}

// HashtagPostListResponse is a cursor-paginated list of posts for a hashtag.
type HashtagPostListResponse struct {
	Hashtag    string         `json:"hashtag"`
	Data       []PostResponse `json:"data"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// HashtagSearchResponse is the response for the hashtag prefix-search endpoint.
type HashtagSearchResponse struct {
	Prefix  string   `json:"prefix"`
	Results []string `json:"results"`
}

// ToTrendingHashtagListResponse converts trending hashtag entities to the response type.
func ToTrendingHashtagListResponse(tags []*entity.TrendingHashtag) TrendingHashtagListResponse {
	data := make([]TrendingHashtagResponse, len(tags))
	for i, t := range tags {
		data[i] = TrendingHashtagResponse{Name: t.Name, Count: t.Count}
	}
	return TrendingHashtagListResponse{Data: data}
}

// ToHashtagPostListResponse builds a cursor-paginated post list response for a hashtag.
func ToHashtagPostListResponse(name string, posts []*entity.Post, nextCursor *post.UserPostCursor, store storage.Storage) HashtagPostListResponse {
	data := make([]PostResponse, len(posts))
	for i, p := range posts {
		data[i] = ToPostResponse(p, store)
	}
	resp := HashtagPostListResponse{Hashtag: name, Data: data}
	if nextCursor != nil {
		resp.NextCursor = nextCursor.Encode()
	}
	return resp
}
