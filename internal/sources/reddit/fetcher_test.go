package reddit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	goreddit "github.com/vartanbeno/go-reddit/v2/reddit"
)

func TestComputeRetryDelay_RateLimitRetryAfterHeader(t *testing.T) {
	err := &goreddit.RateLimitError{
		Response: &http.Response{Header: http.Header{"Retry-After": []string{"7"}}},
	}

	delay, retryable, reason := computeRetryDelay(err, 1)
	if !retryable {
		t.Fatalf("expected retryable=true")
	}
	if reason != "reddit_rate_limit_retry_after" {
		t.Fatalf("expected reason reddit_rate_limit_retry_after, got %q", reason)
	}
	if delay != 7*time.Second {
		t.Fatalf("expected delay 7s, got %s", delay)
	}
}

func TestComputeRetryDelay_RateLimitResetFallback(t *testing.T) {
	reset := time.Now().Add(4 * time.Second)
	err := &goreddit.RateLimitError{
		Rate: goreddit.Rate{Reset: reset},
	}

	delay, retryable, reason := computeRetryDelay(err, 1)
	if !retryable {
		t.Fatalf("expected retryable=true")
	}
	if reason != "reddit_rate_limit_reset" {
		t.Fatalf("expected reason reddit_rate_limit_reset, got %q", reason)
	}
	if delay < 3*time.Second || delay > 4*time.Second+500*time.Millisecond {
		t.Fatalf("expected delay around 4s, got %s", delay)
	}
}

func TestComputeRetryDelay_HTTP429UsesXRateLimitReset(t *testing.T) {
	err := &goreddit.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"X-Ratelimit-Reset": []string{"11"}},
		},
	}

	delay, retryable, reason := computeRetryDelay(err, 2)
	if !retryable {
		t.Fatalf("expected retryable=true")
	}
	if reason != "http_429_x_ratelimit_reset" {
		t.Fatalf("expected reason http_429_x_ratelimit_reset, got %q", reason)
	}
	if delay != 11*time.Second {
		t.Fatalf("expected delay 11s, got %s", delay)
	}
}

func TestComputeRetryDelay_OAuth429Fallback(t *testing.T) {
	err := fmt.Errorf("oauth2: cannot fetch token: 429 Too Many Requests")

	delay, retryable, reason := computeRetryDelay(err, 1)
	if !retryable {
		t.Fatalf("expected retryable=true")
	}
	if reason != "oauth_token_429" {
		t.Fatalf("expected reason oauth_token_429, got %q", reason)
	}
	if delay < 15*time.Second {
		t.Fatalf("expected delay >= 15s, got %s", delay)
	}
}

func TestComputeRetryDelay_NonRetryable(t *testing.T) {
	delay, retryable, reason := computeRetryDelay(fmt.Errorf("validation failed"), 1)
	if retryable {
		t.Fatalf("expected retryable=false")
	}
	if delay != 0 {
		t.Fatalf("expected zero delay for non-retryable errors, got %s", delay)
	}
	if reason != "non_retryable_error" {
		t.Fatalf("expected reason non_retryable_error, got %q", reason)
	}
}

func TestNewSurfHTTPClient_WithProxyURL(t *testing.T) {
	t.Parallel()

	proxyURL, err := url.Parse("http://user:pass@proxy.example.com:12321")
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}

	client := newSurfHTTPClient(nil, 5*time.Second, nil, proxyURL)
	if client == nil {
		t.Fatalf("expected client, got nil")
	}
	if _, ok := client.Transport.(*observabilityRoundTripper); !ok {
		t.Fatalf("expected observabilityRoundTripper transport, got %T", client.Transport)
	}
}

func TestNewHTTPClient_WrapsObservabilityTransport(t *testing.T) {
	t.Parallel()

	client := newSurfHTTPClient(nil, 5*time.Second, nil, nil)
	if client.Timeout != 5*time.Second {
		t.Fatalf("unexpected timeout: %s", client.Timeout)
	}
	if _, ok := client.Transport.(*observabilityRoundTripper); !ok {
		t.Fatalf("expected observabilityRoundTripper transport, got %T", client.Transport)
	}
}

func TestRewriteHTMLAPIError_UsesMessageWhenBodyConsumed(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodGet, "https://www.reddit.com/comments/1r8snay.json", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	apiErr := &goreddit.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
			Request:    req,
			Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
			Body:       io.NopCloser(strings.NewReader("")),
		},
		Message: "<html><body><h1>Blocked</h1><p>Access denied</p></body></html>",
	}

	rewrittenErr, rewritten := rewriteHTMLAPIError(apiErr)
	if !rewritten {
		t.Fatalf("expected HTML error to be rewritten")
	}
	if rewrittenErr == nil {
		t.Fatalf("expected rewritten error")
	}
	if strings.Contains(rewrittenErr.Error(), "<html>") {
		t.Fatalf("expected rewritten error to omit raw HTML, got %q", rewrittenErr.Error())
	}
	if !strings.Contains(rewrittenErr.Error(), "Blocked") {
		t.Fatalf("expected rewritten error to include converted content, got %q", rewrittenErr.Error())
	}
	if !strings.Contains(rewrittenErr.Error(), "403 server returned HTML error") {
		t.Fatalf("expected rewritten error to include status context, got %q", rewrittenErr.Error())
	}
}

func TestDoWithRetry_RewritesNonRetryableHTMLErrors(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodGet, "https://www.reddit.com/comments/1r8snay.json", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	fetcher := &RedditFetcher{logger: slog.Default()}
	err = fetcher.doWithRetry(context.Background(), "fetch_comments", func() error {
		return &goreddit.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
				Request:    req,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			Message: "<html><body><h1>Forbidden</h1><p>Denied</p></body></html>",
		}
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), "<html>") {
		t.Fatalf("expected final retry error to omit raw HTML, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Forbidden") {
		t.Fatalf("expected final retry error to include converted markdown, got %q", err.Error())
	}
}
