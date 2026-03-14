package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
)

const defaultBaseURL = "http://localhost:8000"

// Reader loads PDF content through a Docling conversion service and returns the
// extracted text or markdown so it can be consumed by existing source processors.
type Reader struct {
	client      *http.Client
	baseURL     string
	maxBodySize int64
}

// NewReader builds a Docling-backed reader with conservative defaults so arXiv
// PDF parsing can work without additional per-call setup.
func NewReader(timeout time.Duration, baseURL string) *Reader {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Reader{
		client:      &http.Client{Timeout: timeout},
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		maxBodySize: 10 << 20, // 10 MiB
	}
}

// Read submits the target document URL to Docling's `/convert` endpoint and
// normalizes the response into a plain string.
func (r *Reader) Read(ctx context.Context, urlStr string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "", fmt.Errorf("docling: url is required")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("docling: invalid url: %w", err)
	}
	if isBlocklistedURL(parsedURL) {
		return "", fmt.Errorf("docling: url is blocklisted")
	}

	endpoint, err := buildConvertURL(r.baseURL, urlStr)
	if err != nil {
		return "", fmt.Errorf("docling: build convert url: %w", err)
	}

	var (
		lastStatus int
		respBody   []byte
		respType   string
	)
	err = retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}

		resp, err := r.client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		limited := io.LimitReader(resp.Body, r.maxBodySize+1)
		body, err := io.ReadAll(limited)
		if err != nil {
			return err
		}
		if int64(len(body)) > r.maxBodySize {
			return fmt.Errorf("docling: response too large")
		}

		lastStatus = resp.StatusCode
		respBody = body
		respType = resp.Header.Get("Content-Type")

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("docling transient error: %s", resp.Status)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("docling request failed: %s", resp.Status)
		}
		return nil
	})
	if err != nil {
		msg := strings.TrimSpace(string(respBody))
		if msg != "" {
			msg = ": " + msg
		}
		if lastStatus != 0 {
			return "", fmt.Errorf("docling: status %d%s: %w", lastStatus, msg, err)
		}
		return "", fmt.Errorf("docling: %w", err)
	}

	content, err := extractContent(respType, respBody)
	if err != nil {
		return "", fmt.Errorf("docling: %w", err)
	}
	return strings.TrimSpace(content), nil
}

// buildConvertURL mirrors the curl example and issues a GET request with the
// target PDF URL encoded as the `url` query parameter.
func buildConvertURL(baseURL string, targetURL string) (string, error) {
	parsedBase, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	parsedBase.Path = strings.TrimRight(parsedBase.Path, "/") + "/convert"
	query := parsedBase.Query()
	query.Set("url", targetURL)
	parsedBase.RawQuery = query.Encode()
	return parsedBase.String(), nil
}

// extractContent accepts either plain text or a JSON payload and extracts the
// first meaningful text field from common Docling-style response envelopes.
func extractContent(contentType string, body []byte) (string, error) {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return "", fmt.Errorf("empty response body")
	}

	if !looksLikeJSON(contentType, raw) {
		return raw, nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		// Some deployments may mislabel a plain text body as JSON; in that case
		// preserve the raw content rather than failing the read entirely.
		return raw, nil
	}

	content := findBestContent(payload)
	if content == "" {
		return "", fmt.Errorf("no content found in response")
	}
	return content, nil
}

func looksLikeJSON(contentType string, raw string) bool {
	if strings.Contains(strings.ToLower(contentType), "json") {
		return true
	}
	return strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, `"`)
}

// findBestContent recursively searches the parsed payload for likely content
// fields, preferring markdown/text keys before falling back to arrays.
func findBestContent(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if content := findBestContent(item); content != "" {
				parts = append(parts, content)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n\n"))
	case map[string]any:
		for _, key := range []string{
			"markdown",
			"content",
			"text",
			"body",
			"result",
			"raw_markdown",
			"fit_markdown",
			"document",
			"doc",
			"data",
			"pages",
			"chunks",
		} {
			if nested, ok := typed[key]; ok {
				if content := findBestContent(nested); content != "" {
					return content
				}
			}
		}

		// Sort the remaining keys to keep fallback extraction deterministic.
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if content := findBestContent(typed[key]); content != "" {
				return content
			}
		}
	}
	return ""
}

func isBlocklistedURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, blocked := range []string{"localhost"} {
		if host == blocked {
			return true
		}
	}
	return false
}
