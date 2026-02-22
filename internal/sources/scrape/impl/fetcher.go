package impl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
	"github.com/bakkerme/curator-ai/internal/sources/scrape"
)

// Fetcher implements scrape.Fetcher over net/http.
type Fetcher struct {
	client    *http.Client
	userAgent string
}

func NewFetcher(timeout time.Duration, userAgent string) *Fetcher {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Fetcher{client: &http.Client{Timeout: timeout}, userAgent: userAgent}
}

func (f *Fetcher) Fetch(ctx context.Context, url string, options scrape.FetchOptions) (string, error) {
	var body string
	err := retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		ua := strings.TrimSpace(options.UserAgent)
		if ua == "" {
			ua = f.userAgent
		}
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		}
		resp, err := f.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("http status %d", resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		body = string(b)
		return nil
	})
	if err != nil {
		return "", err
	}
	return body, nil
}
