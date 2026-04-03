package dto

import (
	"github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/pagination"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// CreateCommentRequest is the body for creating a comment or reply
type CreateCommentRequest struct {
	Content        string   `json:"content"          example:"Great post!"`
	ParentID       *string  `json:"parent_id"        example:"null"`
	MediaKeys      []string `json:"media_keys"       example:"[\"media/abc123.jpg\"]"`
	MentionUserIDs []string `json:"mention_user_ids" example:"[\"uuid-of-alice\"]"`
}

// CommentResponse is the outgoing comment representation
type CommentResponse struct {
	ID         string            `json:"id"`
	PostID     string            `json:"post_id"`
	AuthorID   string            `json:"author_id"`
	Author     *AuthorResponse   `json:"author,omitempty"`
	ParentID   *string           `json:"parent_id,omitempty"`
	Content    string            `json:"content"`
	Media      []MediaResponse   `json:"media"`
	Mentions   []MentionResponse `json:"mentions"`
	LikeCount  int64             `json:"like_count"`
	IsLiked    bool              `json:"is_liked"`
	ReplyCount int64             `json:"reply_count"`
	CreatedAt  string            `json:"created_at"`
	Replies    []CommentResponse `json:"replies,omitempty"`
}

// CommentListResponse is a paginated list of comments
type CommentListResponse struct {
	Data []CommentResponse `json:"data"`
	pagination.PaginationResponse
}

// ToCommentResponse converts a comment entity to CommentResponse
func ToCommentResponse(c *entity.Comment, store storage.Storage) CommentResponse {
	media := make([]MediaResponse, len(c.Media))
	for i, m := range c.Media {
		media[i] = MediaResponse{
			ID:        m.ID.String(),
			MediaType: m.MediaType,
			URL:       store.URL(m.MediaKey),
			Position:  m.Position,
		}
	}
	mentions := make([]MentionResponse, len(c.Mentions))
	for i, m := range c.Mentions {
		mentions[i] = MentionResponse{
			ID:          m.ID.String(),
			Username:    m.Username,
			DisplayName: m.DisplayName,
		}
	}

	resp := CommentResponse{
		ID:         c.ID.String(),
		PostID:     c.PostID.String(),
		AuthorID:   c.AuthorID.String(),
		Content:    c.Content,
		Media:      media,
		Mentions:   mentions,
		LikeCount:  c.LikeCount,
		IsLiked:    c.IsLiked,
		ReplyCount: c.ReplyCount,
		CreatedAt:  c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if c.ParentID != nil {
		s := c.ParentID.String()
		resp.ParentID = &s
	}
	if c.Author != nil {
		a := &AuthorResponse{
			ID:          c.Author.ID.String(),
			Username:    c.Author.Username,
			DisplayName: c.Author.DisplayName,
		}
		if c.Author.AvatarKey != nil {
			url := store.URL(*c.Author.AvatarKey)
			a.AvatarURL = &url
		}
		resp.Author = a
	}
	if len(c.Replies) > 0 {
		resp.Replies = make([]CommentResponse, len(c.Replies))
		for i, r := range c.Replies {
			resp.Replies[i] = ToCommentResponse(r, store)
		}
	}
	return resp
}

// ToCommentListResponse builds a paginated comment list response
func ToCommentListResponse(comments []*entity.Comment, pag pagination.PaginationResponse, store storage.Storage) CommentListResponse {
	data := make([]CommentResponse, len(comments))
	for i, c := range comments {
		data[i] = ToCommentResponse(c, store)
	}
	return CommentListResponse{Data: data, PaginationResponse: pag}
}
