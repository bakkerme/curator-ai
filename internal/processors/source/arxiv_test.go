package source

import (
	"context"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/sources/arxiv"
	"github.com/bakkerme/curator-ai/internal/sources/jina"
)

type arxivFetcherMock struct {
	papers []arxiv.Paper
}

func (m *arxivFetcherMock) Search(ctx context.Context, options arxiv.SearchOptions) ([]arxiv.Paper, error) {
	_ = ctx
	_ = options
	return m.papers, nil
}

type arxivReaderMock struct {
	pages map[string]string
	calls []string
}

func (m *arxivReaderMock) Read(ctx context.Context, url string, options jina.ReadOptions) (string, error) {
	_ = ctx
	_ = options
	m.calls = append(m.calls, url)
	return m.pages[url], nil
}

func TestArxivProcessor_AbstractOnlyUsesAbstractContent(t *testing.T) {
	abstractOnly := true
	cfg := &config.ArxivSource{
		Query:        "llm security",
		AbstractOnly: &abstractOnly,
		SummaryPlan:  &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &arxivFetcherMock{
		papers: []arxiv.Paper{{
			ID:          "1234.5678",
			Title:       "Paper",
			Abstract:    "This is only the abstract.",
			Authors:     []string{"A", "B"},
			PublishedAt: time.Now().UTC(),
			AbsURL:      "https://arxiv.org/abs/1234.5678",
			HTMLURL:     "https://arxiv.org/html/1234.5678",
			PDFURL:      "https://arxiv.org/pdf/1234.5678",
		}},
	}
	reader := &arxivReaderMock{
		pages: map[string]string{
			"https://arxiv.org/html/1234.5678": "full paper content",
		},
	}

	processor, err := NewArxivProcessor(cfg, fetcher, reader, nil, nil)
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
	if blocks[0].Content != "This is only the abstract." {
		t.Fatalf("expected abstract-only content, got %q", blocks[0].Content)
	}
	if len(reader.calls) != 0 {
		t.Fatalf("expected no Jina fetches in abstract_only mode, got %d calls", len(reader.calls))
	}
}

func TestArxivProcessor_DefaultModeFetchesFullText(t *testing.T) {
	cfg := &config.ArxivSource{
		Query:       "llm security",
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModeFull},
	}
	fetcher := &arxivFetcherMock{
		papers: []arxiv.Paper{{
			ID:          "1234.5678",
			Title:       "Paper",
			Abstract:    "short abstract",
			Authors:     []string{"A"},
			PublishedAt: time.Now().UTC(),
			AbsURL:      "https://arxiv.org/abs/1234.5678",
			HTMLURL:     "https://arxiv.org/html/1234.5678",
		}},
	}
	reader := &arxivReaderMock{
		pages: map[string]string{
			"https://arxiv.org/html/1234.5678": "full paper content",
		},
	}

	processor, err := NewArxivProcessor(cfg, fetcher, reader, nil, nil)
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
	if blocks[0].Content != "full paper content" {
		t.Fatalf("expected full text content, got %q", blocks[0].Content)
	}
	if len(reader.calls) != 1 {
		t.Fatalf("expected 1 Jina fetch in default mode, got %d calls", len(reader.calls))
	}
}

