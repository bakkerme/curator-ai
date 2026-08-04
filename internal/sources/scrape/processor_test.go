package scrape

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
)

type scrapeFetcherMock struct {
	pages map[string]string
	calls []string
}

func (m *scrapeFetcherMock) Fetch(ctx context.Context, url string, options FetchOptions) (string, error) {
	_ = ctx
	_ = options
	m.calls = append(m.calls, url)
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
		URL:       "https://example.com/blog",
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

func TestScrapeProcessor_MarkdownEnabled_StoresWebBlock(t *testing.T) {
	fetcher := &scrapeFetcherMock{pages: map[string]string{
		"https://example.com/blog": `<html><body><a class="post" href="/p1">p1</a></body></html>`,
		"https://example.com/p1":   `<html><head><title>Post</title></head><body><h1>Post</h1><article><p><strong>Hello</strong> world</p></article></body></html>`,
	}}

	proc, err := NewScrapeProcessor(&config.ScrapeSource{
		URL:       "https://example.com/blog",
		Discovery: config.ScrapeDiscoveryConfig{ItemSelector: ".post", MaxPages: 1},
		Extraction: config.ScrapeExtractionConfig{
			TitleSelector:   "h1",
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
	b := blocks[0]
	if !strings.Contains(b.Content, "**Hello**") {
		t.Fatalf("expected markdown content, got %q", b.Content)
	}
	if len(b.WebBlocks) != 1 {
		t.Fatalf("expected 1 web block, got %d", len(b.WebBlocks))
	}
	wb := b.WebBlocks[0]
	if wb.URL != "https://example.com/p1" {
		t.Fatalf("expected web block URL, got %q", wb.URL)
	}
	if !wb.WasFetched {
		t.Fatal("expected WasFetched=true")
	}
	if !strings.Contains(wb.Page, "<strong>Hello</strong>") {
		t.Fatalf("expected raw HTML in web block page, got %q", wb.Page)
	}
}

func TestScrapeProcessor_MarkdownDisabled_NoWebBlock(t *testing.T) {
	fetcher := &scrapeFetcherMock{pages: map[string]string{
		"https://example.com/blog": `<html><body><a class="post" href="/p1">p1</a></body></html>`,
		"https://example.com/p1":   `<html><head><title>Post</title></head><body><h1>Post</h1><article><p>Hello</p></article></body></html>`,
	}}

	proc, err := NewScrapeProcessor(&config.ScrapeSource{
		URL:       "https://example.com/blog",
		Discovery: config.ScrapeDiscoveryConfig{ItemSelector: ".post", MaxPages: 1},
		Extraction: config.ScrapeExtractionConfig{
			TitleSelector:   "h1",
			ContentSelector: "article",
		},
		Markdown: config.ScrapeMarkdownConfig{Enabled: false},
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
	if len(blocks[0].WebBlocks) != 0 {
		t.Fatalf("expected no web blocks when markdown disabled, got %d", len(blocks[0].WebBlocks))
	}
	if !strings.Contains(blocks[0].Content, "<p>Hello</p>") {
		t.Fatalf("expected raw HTML content, got %q", blocks[0].Content)
	}
}

func TestScrapeProcessor_Pagination(t *testing.T) {
	fetcher := &scrapeFetcherMock{pages: map[string]string{
		"https://example.com/blog": `<html><body>
<a class="post" href="/p1">p1</a>
<a class="next" href="/blog?page=2">next</a>
</body></html>`,
		"https://example.com/blog?page=2": `<html><body>
<a class="post" href="/p2">p2</a>
</body></html>`,
		"https://example.com/p1": `<html><head><title>One</title></head><body><h1>One</h1><article><p>A</p></article></body></html>`,
		"https://example.com/p2": `<html><head><title>Two</title></head><body><h1>Two</h1><article><p>B</p></article></body></html>`,
	}}

	proc, err := NewScrapeProcessor(&config.ScrapeSource{
		URL: "https://example.com/blog",
		Discovery: config.ScrapeDiscoveryConfig{
			ItemSelector:     ".post",
			NextPageSelector: ".next",
			MaxPages:         3,
		},
		Extraction: config.ScrapeExtractionConfig{
			TitleSelector:   "h1",
			ContentSelector: "article",
		},
	}, fetcher, nil, nil)
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}

	blocks, err := proc.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Title != "One" || blocks[1].Title != "Two" {
		t.Fatalf("unexpected titles: %q, %q", blocks[0].Title, blocks[1].Title)
	}
}

func TestScrapeProcessor_FetchDelay(t *testing.T) {
	fetcher := &scrapeFetcherMock{pages: map[string]string{
		"https://example.com/blog": `<html><body><a class="post" href="/p1">p1</a></body></html>`,
		"https://example.com/p1":   `<html><head><title>Post</title></head><body><h1>Post</h1><article><p>A</p></article></body></html>`,
	}}

	proc, err := NewScrapeProcessor(&config.ScrapeSource{
		URL:       "https://example.com/blog",
		Discovery: config.ScrapeDiscoveryConfig{ItemSelector: ".post", MaxPages: 1},
		Extraction: config.ScrapeExtractionConfig{
			TitleSelector:   "h1",
			ContentSelector: "article",
		},
		Request: config.ScrapeRequestConfig{FetchDelay: "10ms"},
	}, fetcher, nil, nil)
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}
	if proc.fetchDelay != 10*time.Millisecond {
		t.Fatalf("expected 10ms fetch delay, got %v", proc.fetchDelay)
	}

	start := time.Now()
	blocks, err := proc.Fetch(context.Background())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if elapsed < 10*time.Millisecond {
		t.Fatalf("expected at least 10ms delay between fetches, elapsed %v", elapsed)
	}
}

func TestScrapeProcessor_InvalidFetchDelay(t *testing.T) {
	_, err := NewScrapeProcessor(&config.ScrapeSource{
		URL:       "https://example.com/blog",
		Discovery: config.ScrapeDiscoveryConfig{ItemSelector: ".post"},
		Extraction: config.ScrapeExtractionConfig{
			ContentSelector: "article",
		},
		Request: config.ScrapeRequestConfig{FetchDelay: "invalid"},
	}, &scrapeFetcherMock{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid fetch_delay")
	}
}

func TestScrapeProcessor_RemoveSelectors(t *testing.T) {
	fetcher := &scrapeFetcherMock{pages: map[string]string{
		"https://example.com/blog": `<html><body><a class="post" href="/p1">p1</a></body></html>`,
		"https://example.com/p1":   `<html><body><h1>Post</h1><article><p>Keep</p><nav>Remove</nav></article></body></html>`,
	}}

	proc, err := NewScrapeProcessor(&config.ScrapeSource{
		URL:       "https://example.com/blog",
		Discovery: config.ScrapeDiscoveryConfig{ItemSelector: ".post", MaxPages: 1},
		Extraction: config.ScrapeExtractionConfig{
			TitleSelector:   "h1",
			ContentSelector: "article",
			RemoveSelectors: []string{"nav"},
		},
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
	if strings.Contains(blocks[0].Content, "Remove") {
		t.Fatalf("expected nav element removed, got %q", blocks[0].Content)
	}
	if !strings.Contains(blocks[0].Content, "Keep") {
		t.Fatalf("expected content to contain Keep, got %q", blocks[0].Content)
	}
}
