package impl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func TestParseFeedLegacyID(t *testing.T) {
	payload := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/hep-th/9901001v2</id>
    <title>Legacy Paper</title>
    <summary>Legacy abstract.</summary>
    <published>1999-01-10T00:00:00Z</published>
    <updated>1999-01-12T00:00:00Z</updated>
    <author><name>John Doe</name></author>
    <category term="hep-th"/>
    <link href="http://arxiv.org/abs/hep-th/9901001v2" rel="alternate" type="text/html"/>
    <link href="http://arxiv.org/pdf/hep-th/9901001v2" rel="related" type="application/pdf"/>
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
	if paper.ID != "hep-th/9901001" {
		t.Fatalf("expected normalized legacy ID, got %q", paper.ID)
	}
	if paper.HTMLURL != "https://arxiv.org/html/hep-th/9901001" {
		t.Fatalf("expected legacy html url, got %q", paper.HTMLURL)
	}
}

func TestNormalizeArxivID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "modern URL",
			raw:  "http://arxiv.org/abs/1234.5678v2",
			want: "1234.5678",
		},
		{
			name: "legacy URL",
			raw:  "http://arxiv.org/abs/hep-th/9901001v2",
			want: "hep-th/9901001",
		},
		{
			name: "legacy bare id",
			raw:  "hep-th/9901001v3",
			want: "hep-th/9901001",
		},
		{
			name: "modern URL with query fragment",
			raw:  "https://arxiv.org/abs/2501.01234v1?context=cs#frag",
			want: "2501.01234",
		},
		{
			name: "do not strip embedded v",
			raw:  "http://arxiv.org/abs/cs.CV/0301001version",
			want: "cs.CV/0301001version",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeArxivID(tc.raw)
			if got != tc.want {
				t.Fatalf("normalizeArxivID(%q): got %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestSearch_DoesNotRetryPermanent4xx(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad query syntax"))
	}))
	defer server.Close()

	fetcher := NewFetcher(2*time.Second, "test-agent", server.URL)
	_, err := fetcher.Search(context.Background(), searchOptions("bad", nil, "", ""))
	if err == nil {
		t.Fatalf("expected search error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected one request without retries, got %d", got)
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status code in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "bad query syntax") {
		t.Fatalf("expected response body details in error, got %v", err)
	}
}

func TestSearch_RetriesTransientStatuses(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call < 3 {
			if call == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limited"))
				return
			}
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("upstream failure"))
			return
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/1234.5678v1</id>
    <title>Sample Paper</title>
    <summary>Abstract text.</summary>
    <published>2024-01-10T00:00:00Z</published>
    <updated>2024-01-12T00:00:00Z</updated>
    <author><name>Jane Doe</name></author>
    <category term="cs.CL"/>
  </entry>
</feed>`))
	}))
	defer server.Close()

	fetcher := NewFetcher(2*time.Second, "test-agent", server.URL)
	papers, err := fetcher.Search(context.Background(), searchOptions("test", nil, "", ""))
	if err != nil {
		t.Fatalf("expected search to succeed after retries, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls (2 retries + success), got %d", got)
	}
	if len(papers) != 1 {
		t.Fatalf("expected one paper, got %d", len(papers))
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
