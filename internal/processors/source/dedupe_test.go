package source

import (
	"context"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
	redditmock "github.com/bakkerme/curator-ai/internal/sources/reddit/mock"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
)

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

func TestRedditProcessorFiltersSeenPosts(t *testing.T) {
	cfg := &config.RedditSource{Subreddits: []string{"golang"}}
	fetcher := &redditmock.Fetcher{
		Items: []reddit.Item{
			{ID: "p1", Title: "seen", URL: "https://example.com/1", CreatedAt: time.Now()},
			{ID: "p2", Title: "new", URL: "https://example.com/2", CreatedAt: time.Now()},
		},
	}
	store := &fakeSeenStore{seen: map[string]bool{"p1": true}}

	processor, err := NewRedditProcessor(cfg, fetcher, nil, store, nil)
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

func TestRSSProcessorFiltersSeenPosts(t *testing.T) {
	cfg := &config.RSSSource{Feeds: []string{"https://example.com/feed.xml"}}
	fetcher := &rssFetcherMock{
		items: []rss.Item{
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
