package dto

import (
	post "github.com/jarviisha/darkvoid/internal/feature/post"
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// CreatePostRequest is the body for creating a new post
type CreatePostRequest struct {
	Content        string   `json:"content"          example:"Hello @alice!"`
	Visibility     string   `json:"visibility"       example:"public"`
	MediaKeys      []string `json:"media_keys"       example:"[\"media/abc123.jpg\"]"`
	MentionUserIDs []string `json:"mention_user_ids" example:"[\"uuid-of-alice\"]"`
	Tags           []string `json:"tags"             example:"[\"golang\",\"webdev\"]"`
}

// UpdatePostRequest is the body for updating a post
type UpdatePostRequest struct {
	Content        string   `json:"content"          example:"Updated content"`
	Visibility     string   `json:"visibility"       example:"followers"`
	MentionUserIDs []string `json:"mention_user_ids" example:"[\"uuid-of-alice\"]"`
	Tags           []string `json:"tags"             example:"[\"golang\",\"webdev\"]"`
}

// MediaResponse represents one attached media item
type MediaResponse struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
	URL       string `json:"url"`
	Position  int32  `json:"position"`
}

// AuthorResponse holds minimal author info embedded in a post response.
type AuthorResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// MentionResponse holds minimal info for a mentioned user.
type MentionResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// PostResponse is the outgoing post representation
type PostResponse struct {
	ID           string            `json:"id"`
	AuthorID     string            `json:"author_id"`
	Author       *AuthorResponse   `json:"author,omitempty"`
	Content      string            `json:"content"`
	Visibility   string            `json:"visibility"`
	Media        []MediaResponse   `json:"media"`
	Tags         []string          `json:"tags"`
	Mentions     []MentionResponse `json:"mentions"`
	LikeCount    int64             `json:"like_count"`
	CommentCount int64             `json:"comment_count"`
	IsLiked      bool              `json:"is_liked"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// PostListResponse is a cursor-paginated list of posts
type PostListResponse struct {
	Data       []PostResponse `json:"data"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// ToPostResponse converts a post entity to PostResponse
func ToPostResponse(p *entity.Post, store storage.Storage) PostResponse {
	media := make([]MediaResponse, len(p.Media))
	for i, m := range p.Media {
		media[i] = MediaResponse{
			ID:        m.ID.String(),
			MediaType: m.MediaType,
			URL:       store.URL(m.MediaKey),
			Position:  m.Position,
		}
	}
	var author *AuthorResponse
	if p.Author != nil {
		a := &AuthorResponse{
			ID:          p.Author.ID.String(),
			Username:    p.Author.Username,
			DisplayName: p.Author.DisplayName,
		}
		if p.Author.AvatarKey != nil {
			url := store.URL(*p.Author.AvatarKey)
			a.AvatarURL = &url
		}
		author = a
	}

	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}

	mentions := make([]MentionResponse, len(p.Mentions))
	for i, m := range p.Mentions {
		mentions[i] = MentionResponse{
			ID:          m.ID.String(),
			Username:    m.Username,
			DisplayName: m.DisplayName,
		}
	}

	return PostResponse{
		ID:           p.ID.String(),
		AuthorID:     p.AuthorID.String(),
		Author:       author,
		Content:      p.Content,
		Visibility:   string(p.Visibility),
		Media:        media,
		Tags:         tags,
		Mentions:     mentions,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		IsLiked:      p.IsLiked,
		CreatedAt:    p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// ToPostListResponse builds a cursor-paginated post list response
func ToPostListResponse(posts []*entity.Post, nextCursor *post.UserPostCursor, store storage.Storage) PostListResponse {
	data := make([]PostResponse, len(posts))
	for i, p := range posts {
		data[i] = ToPostResponse(p, store)
	}
	resp := PostListResponse{Data: data}
	if nextCursor != nil {
		resp.NextCursor = nextCursor.Encode()
	}
	return resp
}
