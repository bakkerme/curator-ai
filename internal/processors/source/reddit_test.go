package source

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/sources/jina"
	redditmock "github.com/bakkerme/curator-ai/internal/sources/reddit/mock"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
)

type jinaReaderMock struct {
	pages map[string]string
	err   error
	calls []string
}

func (m *jinaReaderMock) Read(ctx context.Context, url string, options jina.ReadOptions) (string, error) {
	_ = ctx
	_ = options
	m.calls = append(m.calls, url)
	if m.err != nil {
		return "", m.err
	}
	return m.pages[url], nil
}

func TestRedditProcessor_IncludeWeb_FetchesWebBlocksViaJina(t *testing.T) {
	cfg := &config.RedditSource{
		Subreddits: []string{"golang"},
		IncludeWeb: true,
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
	reader := &jinaReaderMock{
		pages: map[string]string{"https://example.com/page": "# md"},
	}

	processor, err := NewRedditProcessor(cfg, fetcher, reader, nil)
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

func TestRedditProcessor_IncludeWeb_JinaErrorRecordedOnPost(t *testing.T) {
	cfg := &config.RedditSource{
		Subreddits: []string{"golang"},
		IncludeWeb: true,
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
	reader := &jinaReaderMock{err: fmt.Errorf("boom")}

	processor, err := NewRedditProcessor(cfg, fetcher, reader, nil)
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
