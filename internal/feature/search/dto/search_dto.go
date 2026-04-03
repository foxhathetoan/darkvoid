package dto

// SearchType defines which entity types to search.
type SearchType string

const (
	SearchTypeAll      SearchType = "all"
	SearchTypeUsers    SearchType = "users"
	SearchTypePosts    SearchType = "posts"
	SearchTypeHashtags SearchType = "hashtags"
)

// UserResult is a minimal user representation returned in search results.
type UserResult struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	DisplayName   string  `json:"display_name"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	FollowerCount int64   `json:"follower_count"`
}

// PostResult is a minimal post representation returned in search results.
type PostResult struct {
	ID           string `json:"id"`
	AuthorID     string `json:"author_id"`
	Content      string `json:"content"`
	LikeCount    int64  `json:"like_count"`
	CommentCount int64  `json:"comment_count"`
	CreatedAt    string `json:"created_at"`
}

// SearchResponse is the unified search response body.
type SearchResponse struct {
	Query    string       `json:"query"`
	Type     string       `json:"type"`
	Users    []UserResult `json:"users,omitempty"`
	Posts    []PostResult `json:"posts,omitempty"`
	Hashtags []string     `json:"hashtags,omitempty"`
}
