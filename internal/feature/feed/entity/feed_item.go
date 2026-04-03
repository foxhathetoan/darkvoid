package entity

// Source indicates where a feed item came from.
type Source string

const (
	SourceFollowing Source = "following"
	SourceTrending  Source = "trending"
	SourceDiscover  Source = "discover"
)

// FeedItem is a scored post in the feed.
type FeedItem struct {
	Post   *Post
	Score  float64
	Source Source
}
