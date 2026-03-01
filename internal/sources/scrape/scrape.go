package scrape

import "context"

// FetchOptions controls scrape HTTP behavior.
type FetchOptions struct {
	UserAgent string
}

// Fetcher fetches raw HTML for URLs.
type Fetcher interface {
	Fetch(ctx context.Context, url string, options FetchOptions) (string, error)
}
