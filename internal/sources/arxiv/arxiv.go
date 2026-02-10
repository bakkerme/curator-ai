package arxiv

import (
	"context"
	"time"
)

// SearchOptions defines filters for arXiv API queries.
type SearchOptions struct {
	Query      string
	Categories []string
	MaxResults int
	SortBy     string
	SortOrder  string
	DateFrom   string
	DateTo     string
}

// Paper represents a normalized arXiv API response entry.
type Paper struct {
	ID          string
	Title       string
	Abstract    string
	Authors     []string
	Categories  []string
	PublishedAt time.Time
	UpdatedAt   time.Time
	AbsURL      string
	PDFURL      string
	HTMLURL     string
}

// Fetcher retrieves papers from the arXiv API.
type Fetcher interface {
	Search(ctx context.Context, options SearchOptions) ([]Paper, error)
}
