package source

import (
	"context"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
)

type rssFetcherMock struct {
	items []rss.Item
}

func (m *rssFetcherMock) Fetch(ctx context.Context, feedURL string, options rss.FetchOptions) ([]rss.Item, error) {
	_ = ctx
	_ = feedURL
	return m.items, nil
}

func TestRSSProcessorPrefersContentByDefault(t *testing.T) {
	cfg := &config.RSSSource{
		Feeds: []string{"https://example.com/feed.xml"},
	}
	fetcher := &rssFetcherMock{
		items: []rss.Item{
			{
				ID:          "1",
				Title:       "Test",
				Link:        "https://example.com/post",
				Description: "Summary",
				Content:     "Full content",
			},
		},
	}

	processor, err := NewRSSProcessor(cfg, fetcher)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}
	blocks, err := processor.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Content != "Full content" {
		t.Errorf("expected content to use full content, got %s", blocks[0].Content)
	}
}
