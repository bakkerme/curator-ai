package rss

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

type rssFetcherMock struct {
	items []Item
}

func (m *rssFetcherMock) Fetch(ctx context.Context, feedURL string, options FetchOptions) ([]Item, error) {
	_ = ctx
	_ = feedURL
	return m.items, nil
}

type fakeSeenStore struct {
	seen map[string]bool
	has  []string
	mark []string
	err  error
}

func (s *fakeSeenStore) HasSeen(ctx context.Context, id string) (bool, error) {
	_ = ctx
	s.has = append(s.has, id)
	if s.err != nil {
		return false, s.err
	}
	return s.seen[id], nil
}

func (s *fakeSeenStore) MarkSeen(ctx context.Context, id string) error {
	_ = ctx
	s.mark = append(s.mark, id)
	if s.err != nil {
		return s.err
	}
	s.seen[id] = true
	return nil
}

func (s *fakeSeenStore) MarkSeenBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.MarkSeen(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *fakeSeenStore) Close() error {
	return nil
}

func TestRSSProcessorPrefersContentByDefault(t *testing.T) {
	cfg := &config.RSSSource{
		Feeds:       []string{"https://example.com/feed.xml"},
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &rssFetcherMock{
		items: []Item{
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
		SummaryPlan:             &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &rssFetcherMock{
		items: []Item{
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
		SummaryPlan:             &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &rssFetcherMock{items: []Item{{
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

func TestRSSProcessorFiltersSeenPosts(t *testing.T) {
	cfg := &config.RSSSource{
		Feeds:       []string{"https://example.com/feed.xml"},
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &rssFetcherMock{
		items: []Item{
			{ID: "a", Title: "seen", Link: "https://example.com/a"},
			{ID: "b", Title: "new", Link: "https://example.com/b"},
		},
	}
	store := &fakeSeenStore{seen: map[string]bool{"a": true}}

	processor, err := NewRSSProcessor(cfg, fetcher, store)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	blocks, err := processor.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(blocks) != 1 || blocks[0].ID != "b" {
		t.Fatalf("expected only new post to be emitted")
	}
	if len(store.mark) != 1 || store.mark[0] != "b" {
		t.Fatalf("expected to mark only new post as seen")
	}
}
