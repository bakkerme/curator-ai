package impl

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildCrawlRequest(t *testing.T) {
	t.Parallel()

	req, err := buildCrawlRequest(context.Background(), "http://crawl4ai.test/", "https://example.com/page")
	if err != nil {
		t.Fatalf("buildCrawlRequest() error = %v", err)
	}

	if req.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", req.Method, http.MethodPost)
	}
	if req.URL.String() != "http://crawl4ai.test/crawl" {
		t.Fatalf("url = %q, want %q", req.URL.String(), "http://crawl4ai.test/crawl")
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll(req.Body) error = %v", err)
	}

	var payload map[string][]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got := payload["urls"]; len(got) != 1 || got[0] != "https://example.com/page" {
		t.Fatalf("payload urls = %#v, want %#v", got, []string{"https://example.com/page"})
	}
}

func TestExtractMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "markdown object prefers fit",
			body: `{"success":true,"results":[{"url":"https://example.com/page","markdown":{"raw_markdown":"# raw","fit_markdown":"# fit"},"success":true}]}`,
			want: "# fit",
		},
		{
			name: "markdown object falls back to raw",
			body: `{"success":true,"results":[{"url":"https://example.com/page","markdown":{"raw_markdown":"# raw","fit_markdown":""},"success":true}]}`,
			want: "# raw",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractMarkdown([]byte(tt.body))
			if err != nil {
				t.Fatalf("extractMarkdown() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("extractMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractMarkdownRejectsUnexpectedMarkdownShape(t *testing.T) {
	t.Parallel()

	_, err := extractMarkdown([]byte(`{"success":true,"results":[{"url":"https://example.com/page","markdown":"# hello","success":true}]}`))
	if err == nil {
		t.Fatalf("expected error for string markdown shape")
	}
}

func TestReaderReadMissingURL(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "http://crawl4ai.test")
	_, err := reader.Read(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty url")
	}
}

func TestReaderReadRetriesOnServerError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	reader := NewReader(2*time.Second, server.URL)
	_, err := reader.Read(context.Background(), "https://example.com/page")
	if err == nil {
		t.Fatalf("expected error on 500 response")
	}
	if got := calls.Load(); got < 2 {
		t.Fatalf("expected retry, got %d calls", got)
	}
}
