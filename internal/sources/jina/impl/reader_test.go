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

	"github.com/bakkerme/curator-ai/internal/sources/jina"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReader_Read(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "curator-ai/0.1", "http://jina.test/", "test-key")
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				return nil, fmt.Errorf("method = %s, want %s", r.Method, http.MethodPost)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				return nil, fmt.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				return nil, fmt.Errorf("Content-Type = %q, want %q", got, "application/json")
			}
			if got := r.Header.Get("X-Retain-Images"); got != "none" {
				return nil, fmt.Errorf("X-Retain-Images = %q, want %q", got, "none")
			}
			if got := r.Header.Get("User-Agent"); got != "curator-ai/0.1" {
				return nil, fmt.Errorf("User-Agent = %q, want %q", got, "curator-ai/0.1")
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("read body: %w", err)
			}
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal: %w", err)
			}
			if payload["url"] != "https://example.com/page" {
				return nil, fmt.Errorf("payload url = %q, want %q", payload["url"], "https://example.com/page")
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(" # hello \n")),
				Request:    r,
			}, nil
		}),
	}

	got, err := reader.Read(context.Background(), "https://example.com/page", jina.ReadOptions{})
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != "# hello" {
		t.Fatalf("Read() = %q, want %q", got, "# hello")
	}
}

func TestReader_Read_MissingAPIKey(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "curator-ai/0.1", "http://localhost", "")
	_, err := reader.Read(context.Background(), "https://example.com", jina.ReadOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
