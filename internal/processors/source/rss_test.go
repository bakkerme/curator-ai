package source

import (
	"context"
	"encoding/base64"
	"strings"
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

	processor, err := NewRSSProcessor(cfg, fetcher, nil)
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

func TestRSSProcessorConvertsHTMLToMarkdownWhenEnabled(t *testing.T) {
	cfg := &config.RSSSource{
		Feeds:                   []string{"https://example.com/feed.xml"},
		ConvertSourceToMarkdown: true,
	}
	fetcher := &rssFetcherMock{
		items: []rss.Item{
			{
				ID:          "1",
				Title:       "Test",
				Link:        "https://example.com/post",
				Description: "<p><strong>Bold Text</strong></p>",
			},
		},
	}

	processor, err := NewRSSProcessor(cfg, fetcher, nil)
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
	if blocks[0].Content != "**Bold Text**" {
		t.Fatalf("expected markdown content, got %q", blocks[0].Content)
	}
}

func TestRSSProcessorExtractsDataURIImagesIntoImageBlocks(t *testing.T) {
	// Small (not necessarily valid) PNG-ish byte sequence; we just care that it decodes and is stored.
	imgBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	imgB64 := base64.StdEncoding.EncodeToString(imgBytes)

	cfg := &config.RSSSource{
		Feeds:                   []string{"https://example.com/feed.xml"},
		ConvertSourceToMarkdown: true,
	}
	fetcher := &rssFetcherMock{items: []rss.Item{{
		ID:          "1",
		Title:       "Test",
		Link:        "https://example.com/post",
		Description: "<p>hello<img alt=\"x\" src=\"data:image/png;base64," + imgB64 + "\" /></p>",
	}}}

	processor, err := NewRSSProcessor(cfg, fetcher, nil)
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

	if got := len(blocks[0].ImageBlocks); got != 1 {
		t.Fatalf("expected 1 image block, got %d", got)
	}
	if string(blocks[0].ImageBlocks[0].ImageData) != string(imgBytes) {
		t.Fatalf("expected extracted image bytes to match")
	}

	if strings.Contains(blocks[0].Content, "data:image") {
		t.Fatalf("expected content to be scrubbed of data:image, got %q", blocks[0].Content)
	}
	if !strings.Contains(blocks[0].Content, "curator-image://post/1/0") {
		t.Fatalf("expected placeholder URL in content, got %q", blocks[0].Content)
	}
}
