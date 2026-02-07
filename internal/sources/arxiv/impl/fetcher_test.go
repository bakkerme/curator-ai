package impl

import (
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/sources/arxiv"
)

func TestBuildSearchQuery(t *testing.T) {
	query, err := buildSearchQuery(searchOptions("test query", []string{"cs.CL", "cs.IR"}, "2024-01-01", "2024-01-31"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "all:\"test query\"") {
		t.Fatalf("expected query clause, got %q", query)
	}
	if !strings.Contains(query, "cat:cs.CL") || !strings.Contains(query, "cat:cs.IR") {
		t.Fatalf("expected category clause, got %q", query)
	}
	if !strings.Contains(query, "submittedDate:[") {
		t.Fatalf("expected date clause, got %q", query)
	}
}

func TestParseFeed(t *testing.T) {
	payload := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/1234.5678v2</id>
    <title>Sample Paper</title>
    <summary>Abstract text.</summary>
    <published>2024-01-10T00:00:00Z</published>
    <updated>2024-01-12T00:00:00Z</updated>
    <author><name>Jane Doe</name></author>
    <category term="cs.CL"/>
    <link href="http://arxiv.org/abs/1234.5678v2" rel="alternate" type="text/html"/>
    <link href="http://arxiv.org/pdf/1234.5678v2" rel="related" type="application/pdf"/>
  </entry>
</feed>`)

	entries, err := parseFeed(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	paper := entries[0].toPaper()
	if paper.ID != "1234.5678" {
		t.Fatalf("expected normalized ID, got %q", paper.ID)
	}
	if paper.PDFURL == "" {
		t.Fatalf("expected pdf url")
	}
	if paper.HTMLURL == "" {
		t.Fatalf("expected html url")
	}
}

func searchOptions(query string, categories []string, dateFrom string, dateTo string) arxiv.SearchOptions {
	return arxiv.SearchOptions{
		Query:      query,
		Categories: categories,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
	}
}
