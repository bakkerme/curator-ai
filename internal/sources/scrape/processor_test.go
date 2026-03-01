package scrape

import (
	"context"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
)

type scrapeFetcherMock struct {
	pages map[string]string
}

func (m *scrapeFetcherMock) Fetch(ctx context.Context, url string, options FetchOptions) (string, error) {
	_ = ctx
	_ = options
	return m.pages[url], nil
}

func TestScrapeProcessor_Fetch_WithLookbackAndPostLimit(t *testing.T) {
	now := time.Now().UTC()
	recent := now.Add(-24 * time.Hour).Format(time.RFC3339)
	old := now.Add(-14 * 24 * time.Hour).Format(time.RFC3339)
	fetcher := &scrapeFetcherMock{pages: map[string]string{
		"https://example.com/blog": `<html><body>
<a class="post" href="/p1">p1</a>
<a class="post" href="/p2">p2</a>
</body></html>`,
		"https://example.com/p1": `<html><head><title>One</title></head><body><h1>One</h1><time datetime="` + recent + `"></time><article><p>A</p></article></body></html>`,
		"https://example.com/p2": `<html><head><title>Two</title></head><body><h1>Two</h1><time datetime="` + old + `"></time><article><p>B</p></article></body></html>`,
	}}

	proc, err := NewScrapeProcessor(&config.ScrapeSource{
		URLs:      []string{"https://example.com/blog"},
		PostLimit: 2,
		Lookback:  "7d",
		Discovery: config.ScrapeDiscoveryConfig{ItemSelector: ".post", MaxPages: 1},
		Extraction: config.ScrapeExtractionConfig{
			TitleSelector:   "h1",
			DateSelector:    "time",
			DateAttr:        "datetime",
			ContentSelector: "article",
		},
		Markdown: config.ScrapeMarkdownConfig{Enabled: true},
	}, fetcher, nil, nil)
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}

	blocks, err := proc.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Title != "One" {
		t.Fatalf("expected title One, got %q", blocks[0].Title)
	}
	if blocks[0].Metadata["source_lookback"] != "7d" {
		t.Fatalf("expected source_lookback metadata")
	}
}
