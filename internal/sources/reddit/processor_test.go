package reddit_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
	redditmock "github.com/bakkerme/curator-ai/internal/sources/reddit/mock"
)

type readerMock struct {
	pages map[string]string
	err   error
	calls []string
}

func (m *readerMock) Read(ctx context.Context, url string) (string, error) {
	_ = ctx
	m.calls = append(m.calls, url)
	if m.err != nil {
		return "", m.err
	}
	return m.pages[url], nil
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

func TestRedditProcessor_IncludeWeb_FetchesWebBlocksViaReader(t *testing.T) {
	cfg := &config.RedditSource{
		Subreddits:  []string{"golang"},
		IncludeWeb:  true,
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &redditmock.Fetcher{
		Items: []reddit.Item{
			{
				ID:        "p1",
				Title:     "t",
				URL:       "https://reddit.com/r/golang",
				CreatedAt: time.Now().UTC(),
				WebURLs:   []string{"https://example.com/page"},
			},
		},
	}
	reader := &readerMock{
		pages: map[string]string{"https://example.com/page": "# md"},
	}

	processor, err := reddit.NewRedditProcessor(cfg, fetcher, reader, nil, nil)
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
	if len(blocks[0].WebBlocks) != 1 {
		t.Fatalf("expected 1 web block, got %d", len(blocks[0].WebBlocks))
	}
	if blocks[0].WebBlocks[0].URL != "https://example.com/page" {
		t.Fatalf("web url = %q", blocks[0].WebBlocks[0].URL)
	}
	if !blocks[0].WebBlocks[0].WasFetched {
		t.Fatalf("expected web block to be fetched")
	}
	if blocks[0].WebBlocks[0].Page != "# md" {
		t.Fatalf("web page = %q", blocks[0].WebBlocks[0].Page)
	}
	if len(reader.calls) != 1 {
		t.Fatalf("reader calls = %d, want 1", len(reader.calls))
	}
}

func TestRedditProcessor_IncludeWeb_ReaderErrorRecordedOnPost(t *testing.T) {
	cfg := &config.RedditSource{
		Subreddits:  []string{"golang"},
		IncludeWeb:  true,
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &redditmock.Fetcher{
		Items: []reddit.Item{
			{
				ID:        "p1",
				Title:     "t",
				URL:       "https://reddit.com/r/golang",
				CreatedAt: time.Now().UTC(),
				WebURLs:   []string{"https://example.com/page"},
			},
		},
	}
	reader := &readerMock{err: fmt.Errorf("boom")}

	processor, err := reddit.NewRedditProcessor(cfg, fetcher, reader, nil, nil)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}
	blocks, err := processor.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(blocks) != 1 || len(blocks[0].WebBlocks) != 1 {
		t.Fatalf("expected 1 post with 1 web block")
	}
	if blocks[0].WebBlocks[0].WasFetched {
		t.Fatalf("expected web block to not be fetched")
	}
	if len(blocks[0].Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(blocks[0].Errors))
	}
	if blocks[0].Errors[0].ProcessorName != "reddit" || blocks[0].Errors[0].Stage != "source" {
		t.Fatalf("error metadata = %+v", blocks[0].Errors[0])
	}
	if !strings.Contains(blocks[0].Errors[0].Error, "https://example.com/page") {
		t.Fatalf("error string = %q", blocks[0].Errors[0].Error)
	}
}

func TestRedditProcessorFiltersSeenPosts(t *testing.T) {
	cfg := &config.RedditSource{
		Subreddits:  []string{"golang"},
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &redditmock.Fetcher{
		Items: []reddit.Item{
			{ID: "p1", Title: "seen", URL: "https://example.com/1", CreatedAt: time.Now()},
			{ID: "p2", Title: "new", URL: "https://example.com/2", CreatedAt: time.Now()},
		},
	}
	store := &fakeSeenStore{seen: map[string]bool{"p1": true}}

	processor, err := reddit.NewRedditProcessor(cfg, fetcher, nil, store, nil)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	blocks, err := processor.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(blocks) != 1 || blocks[0].ID != "p2" {
		t.Fatalf("expected only new post to be emitted")
	}
	if len(store.mark) != 1 || store.mark[0] != "p2" {
		t.Fatalf("expected to mark only new post as seen")
	}
}
