package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
)

const defaultBaseURL = "http://localhost:11235"

type Reader struct {
	client      *http.Client
	baseURL     string
	maxBodySize int64
}

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

// buildCrawlRequest creates the Crawl4AI `/crawl` request body so request
// construction can be tested separately from transport behavior.
func buildCrawlRequest(ctx context.Context, baseURL string, targetURL string) (*http.Request, error) {
	payload, err := json.Marshal(map[string][]string{"urls": {targetURL}})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/crawl", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (r *Reader) Read(ctx context.Context, urlStr string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "", fmt.Errorf("crawl4ai: url is required")
	}

	var (
		lastStatus int
		respBody   []byte
	)
	err := retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		req, err := buildCrawlRequest(ctx, r.baseURL, urlStr)
		if err != nil {
			return retry.Permanent(err)
		}

		resp, err := r.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		limited := io.LimitReader(resp.Body, r.maxBodySize+1)
		body, err := io.ReadAll(limited)
		if err != nil {
			return err
		}
		if int64(len(body)) > r.maxBodySize {
			return fmt.Errorf("crawl4ai: response too large")
		}

		lastStatus = resp.StatusCode
		respBody = body

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("crawl4ai transient error: %s", resp.Status)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("crawl4ai request failed: %s", resp.Status)
		}
		return nil
	})
	if err != nil {
		msg := strings.TrimSpace(string(respBody))
		if msg != "" {
			msg = ": " + msg
		}
		if lastStatus != 0 {
			return "", fmt.Errorf("crawl4ai: status %d%s: %w", lastStatus, msg, err)
		}
		return "", fmt.Errorf("crawl4ai: %w", err)
	}

	md, err := extractMarkdown(respBody)
	if err != nil {
		return "", fmt.Errorf("crawl4ai: %w", err)
	}
	return strings.TrimSpace(md), nil
}

type crawlResponse struct {
	Success bool          `json:"success"`
	Results []crawlResult `json:"results"`
}

type crawlResult struct {
	URL      string         `json:"url"`
	Markdown markdownObject `json:"markdown"`
	Success  bool           `json:"success"`
}

type markdownObject struct {
	FitMarkdown string `json:"fit_markdown"`
	RawMarkdown string `json:"raw_markdown"`
}

// extractMarkdown returns the best markdown field from the first Crawl4AI
// result, preferring fit_markdown over raw_markdown when both are present.
func extractMarkdown(body []byte) (string, error) {
	var cr crawlResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(cr.Results) == 0 {
		return "", fmt.Errorf("no results in response")
	}

	res := cr.Results[0]
	if res.Markdown.FitMarkdown != "" {
		return res.Markdown.FitMarkdown, nil
	}
	if res.Markdown.RawMarkdown != "" {
		return res.Markdown.RawMarkdown, nil
	}
	return "", fmt.Errorf("no markdown in response")
}
