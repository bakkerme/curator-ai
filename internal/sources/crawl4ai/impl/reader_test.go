package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"


)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonBody(v any) io.ReadCloser {
	b, _ := json.Marshal(v)
	return io.NopCloser(strings.NewReader(string(b)))
}

func crawlResponseBody(markdown string) io.ReadCloser {
	return jsonBody(map[string]any{
		"success": true,
		"results": []map[string]any{
			{"url": "https://example.com/page", "markdown": markdown, "success": true},
		},
	})
}

func TestReader_Read(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "http://crawl4ai.test")
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				return nil, fmt.Errorf("method = %s, want %s", r.Method, http.MethodPost)
			}
			if got := r.URL.Path; got != "/crawl" {
				return nil, fmt.Errorf("path = %q, want %q", got, "/crawl")
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				return nil, fmt.Errorf("Content-Type = %q, want %q", got, "application/json")
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("read body: %w", err)
			}
			var payload map[string][]string
			if err := json.Unmarshal(body, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal: %w", err)
			}
			if len(payload["urls"]) == 0 || payload["urls"][0] != "https://example.com/page" {
				return nil, fmt.Errorf("payload urls[0] = %q, want %q", payload["urls"][0], "https://example.com/page")
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       crawlResponseBody("# hello"),
				Request:    r,
			}, nil
		}),
	}

	got, err := reader.Read(context.Background(), "https://example.com/page")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != "# hello" {
		t.Fatalf("Read() = %q, want %q", got, "# hello")
	}
}

func TestReader_Read_MarkdownObject(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "http://crawl4ai.test")
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body := jsonBody(map[string]any{
				"success": true,
				"results": []map[string]any{
					{
						"url": "https://example.com/page",
						"markdown": map[string]string{
							"raw_markdown": "# raw",
							"fit_markdown": "# fit",
						},
						"success": true,
					},
				},
			})
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       body,
				Request:    r,
			}, nil
		}),
	}

	got, err := reader.Read(context.Background(), "https://example.com/page")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != "# fit" {
		t.Fatalf("Read() = %q, want %q", got, "# fit")
	}
}

func TestReader_Read_MissingURL(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "http://crawl4ai.test")
	_, err := reader.Read(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty url")
	}
}

func TestReader_Read_ErrorResponse(t *testing.T) {
	t.Parallel()

	calls := 0
	reader := NewReader(2*time.Second, "http://crawl4ai.test")
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("server error")),
				Request:    r,
			}, nil
		}),
	}

	_, err := reader.Read(context.Background(), "https://example.com/page")
	if err == nil {
		t.Fatalf("expected error on 500 response")
	}
	if calls < 2 {
		t.Fatalf("expected retry, got %d calls", calls)
	}
}
