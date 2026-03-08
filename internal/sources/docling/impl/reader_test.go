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

func TestReader_Read_PlainText(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "http://docling.test")
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				return nil, fmt.Errorf("method = %s, want %s", r.Method, http.MethodGet)
			}
			if got := r.URL.Path; got != "/convert" {
				return nil, fmt.Errorf("path = %q, want %q", got, "/convert")
			}
			if got := r.URL.Query().Get("url"); got != "https://arxiv.org/pdf/1234.5678" {
				return nil, fmt.Errorf("url query = %q, want %q", got, "https://arxiv.org/pdf/1234.5678")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader("# converted markdown")),
				Request:    r,
			}, nil
		}),
	}

	got, err := reader.Read(context.Background(), "https://arxiv.org/pdf/1234.5678")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != "# converted markdown" {
		t.Fatalf("Read() = %q, want %q", got, "# converted markdown")
	}
}

func TestReader_Read_JSONEnvelope(t *testing.T) {
	t.Parallel()

	reader := NewReader(2*time.Second, "http://docling.test")
	reader.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(map[string]any{
				"document": map[string]any{
					"markdown": "# parsed",
				},
			})
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Request:    r,
			}, nil
		}),
	}

	got, err := reader.Read(context.Background(), "https://arxiv.org/pdf/1234.5678")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != "# parsed" {
		t.Fatalf("Read() = %q, want %q", got, "# parsed")
	}
}

func TestReader_Read_ErrorResponseRetries(t *testing.T) {
	t.Parallel()

	calls := 0
	reader := NewReader(2*time.Second, "http://docling.test")
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

	_, err := reader.Read(context.Background(), "https://arxiv.org/pdf/1234.5678")
	if err == nil {
		t.Fatalf("expected error on 500 response")
	}
	if calls < 2 {
		t.Fatalf("expected retry, got %d calls", calls)
	}
}

func TestBuildConvertURL(t *testing.T) {
	t.Parallel()

	got, err := buildConvertURL("http://docling.test", "https://arxiv.org/pdf/2602.05868")
	if err != nil {
		t.Fatalf("buildConvertURL() error = %v", err)
	}

	want := "http://docling.test/convert?url=https%3A%2F%2Farxiv.org%2Fpdf%2F2602.05868"
	if got != want {
		t.Fatalf("buildConvertURL() = %q, want %q", got, want)
	}
}
