package reddit

import (
	"context"
	"time"
)

// Config describes the reddit fetch configuration.
type Config struct {
	Subreddits      []string
	Limit           int
	Sort            string
	TimeFilter      string
	IncludeComments bool
	IncludeWeb      bool
	IncludeImages   bool
	MinScore        int
	UserAgent       string
}

// Comment represents a single reddit comment.
type Comment struct {
	ID        string
	Author    string
	Content   string
	CreatedAt time.Time
}

// Item represents a single reddit post.
type Item struct {
	ID        string
	Title     string
	URL       string
	Content   string
	Author    string
	Score     int
	CreatedAt time.Time
	Comments  []Comment
	WebURLs   []string
	ImageURLs []string
}

// Fetcher retrieves reddit posts based on config.
type Fetcher interface {
	Fetch(ctx context.Context, config Config) ([]Item, error)
}
