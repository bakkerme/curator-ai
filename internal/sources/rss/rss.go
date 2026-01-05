package rss

import (
	"context"
	"time"
)

// FetchOptions controls RSS fetch behavior.
type FetchOptions struct {
	Limit     int
	UserAgent string
}

// Item represents a single RSS or Atom entry.
type Item struct {
	ID          string
	Title       string
	Link        string
	Description string
	Content     string
	Author      string
	PublishedAt time.Time
}

// Fetcher fetches and parses RSS/Atom feeds.
type Fetcher interface {
	Fetch(ctx context.Context, feedURL string, options FetchOptions) ([]Item, error)
}
