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

// Item represents a single reddit post.
type Item struct {
	ID        string
	Title     string
	URL       string
	Content   string
	Author    string
	Score     int
	CreatedAt time.Time
}

// Fetcher retrieves reddit posts based on config.
type Fetcher interface {
	Fetch(ctx context.Context, config Config) ([]Item, error)
}
