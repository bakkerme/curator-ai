package reddit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	goreddit "github.com/vartanbeno/go-reddit/v2/reddit"
)

type RedditFetcher struct {
	client         *goreddit.Client
	initErr        error
	logger         *slog.Logger
	requestCounter atomic.Uint64
}

type observabilityRoundTripper struct {
	base    http.RoundTripper
	logger  *slog.Logger
	counter *atomic.Uint64
}

func (t *observabilityRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	total := t.counter.Add(1)
	resp, err := base.RoundTrip(req)
	if err != nil {
		t.logger.Info(
			"Reddit HTTP request failed",
			slog.Uint64("total_requests", total),
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	t.logger.Info(
		"Reddit HTTP request completed",
		slog.Uint64("total_requests", total),
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Int("status_code", resp.StatusCode),
		slog.String("ratelimit_remaining", resp.Header.Get("x-ratelimit-remaining")),
		slog.String("ratelimit_used", resp.Header.Get("x-ratelimit-used")),
		slog.String("ratelimit_reset", resp.Header.Get("x-ratelimit-reset")),
		slog.String("retry_after", resp.Header.Get("retry-after")),
	)

	return resp, nil
}

const (
	redditRetryAttempts  = 3
	redditRetryBaseDelay = 200 * time.Millisecond
	redditRetryMaxDelay  = 2 * time.Minute
)

func NewFetcher(logger *slog.Logger, timeout time.Duration, userAgent, clientID, clientSecret, username, password string, proxyURL *url.URL) Fetcher {
	if logger == nil {
		logger = slog.Default()
	}

	fetcher := &RedditFetcher{logger: logger}
	httpClient := newSurfHTTPClient(logger, timeout, &fetcher.requestCounter, proxyURL)

	var (
		client *goreddit.Client
		err    error
	)
	if clientID != "" && clientSecret != "" && username != "" && password != "" {
		logger.Info("Using authenticated Reddit client", slog.String("clientID", clientID))
		client, err = goreddit.NewClient(goreddit.Credentials{
			ID:       clientID,
			Secret:   clientSecret,
			Username: username,
			Password: password,
		}, goreddit.WithHTTPClient(httpClient), goreddit.WithUserAgent(userAgent))
	} else {
		logger.Info("Using readonly Reddit client")
		client, err = goreddit.NewReadonlyClient(goreddit.WithHTTPClient(httpClient), goreddit.WithUserAgent(userAgent))
	}

	fetcher.client = client
	fetcher.initErr = err
	return fetcher
}

func (f *RedditFetcher) Fetch(ctx context.Context, config Config) ([]Item, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}
	if len(config.Subreddits) == 0 {
		return nil, fmt.Errorf("no subreddits configured")
	}

	sort := config.Sort
	if sort == "" {
		sort = "hot"
	}

	limit := config.Limit
	if limit <= 0 {
		limit = 25
	}

	requestsBefore := f.requestCounter.Load()

	f.logger.Info("Fetching Reddit posts", slog.String("subreddits", strings.Join(config.Subreddits, ",")), slog.String("sort", sort), slog.Int("limit", limit))
	subreddits := strings.Join(config.Subreddits, "+")
	posts, err := f.fetchPosts(ctx, subreddits, sort, limit, config.TimeFilter)
	if err != nil {
		return nil, fmt.Errorf("got error fetching reddit posts %w", err)
	}

	items := make([]Item, 0, len(posts))
	for _, post := range posts {
		if post == nil {
			continue
		}
		if config.MinScore > 0 && post.Score < config.MinScore {
			continue
		}

		item := Item{
			ID:        post.ID,
			Title:     post.Title,
			URL:       canonicalRedditPostURL(post.Permalink),
			Content:   post.Body,
			Author:    post.Author,
			Score:     post.Score,
			CreatedAt: timestampToTime(post.Created),
		}

		if config.IncludeWeb || config.IncludeImages {
			item.WebURLs, item.ImageURLs = extractPostURLs(post)
			if !config.IncludeWeb {
				item.WebURLs = nil
			}
			if !config.IncludeImages {
				item.ImageURLs = nil
			}
		}

		if config.IncludeComments {
			comments, err := f.fetchTopLevelComments(ctx, post.ID, 25)
			if err != nil {
				return nil, err
			}
			item.Comments = comments
		}

		items = append(items, item)
	}

	f.logger.Info(
		"Finished Reddit fetch",
		slog.Int("posts_fetched", len(items)),
		slog.Int("http_requests_this_fetch", int(f.requestCounter.Load()-requestsBefore)),
		slog.Uint64("http_requests_total", f.requestCounter.Load()),
	)
	return items, nil
}

func (f *RedditFetcher) fetchPosts(ctx context.Context, subreddit, sort string, limit int, timeFilter string) ([]*goreddit.Post, error) {
	var posts []*goreddit.Post
	err := f.doWithRetry(ctx, "fetch_posts", func() error {
		var err error
		switch strings.ToLower(sort) {
		case "hot":
			posts, _, err = f.client.Subreddit.HotPosts(ctx, subreddit, &goreddit.ListOptions{Limit: limit})
		case "new":
			posts, _, err = f.client.Subreddit.NewPosts(ctx, subreddit, &goreddit.ListOptions{Limit: limit})
		case "rising":
			posts, _, err = f.client.Subreddit.RisingPosts(ctx, subreddit, &goreddit.ListOptions{Limit: limit})
		case "top":
			posts, _, err = f.client.Subreddit.TopPosts(ctx, subreddit, &goreddit.ListPostOptions{
				ListOptions: goreddit.ListOptions{Limit: limit},
				Time:        timeFilter,
			})
		case "controversial":
			posts, _, err = f.client.Subreddit.ControversialPosts(ctx, subreddit, &goreddit.ListPostOptions{
				ListOptions: goreddit.ListOptions{Limit: limit},
				Time:        timeFilter,
			})
		default:
			return fmt.Errorf("unsupported reddit sort: %q", sort)
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (f *RedditFetcher) fetchTopLevelComments(ctx context.Context, postID string, limit int) ([]Comment, error) {
	if limit <= 0 {
		return nil, nil
	}

	var (
		pc *goreddit.PostAndComments
	)
	err := f.doWithRetry(ctx, "fetch_comments", func() error {
		var err error
		pc, _, err = f.client.Post.Get(ctx, postID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if pc == nil || len(pc.Comments) == 0 {
		return nil, nil
	}

	parentFullID := "t3_" + postID
	out := make([]Comment, 0, min(len(pc.Comments), limit))
	for _, c := range pc.Comments {
		if c == nil || c.ParentID != parentFullID {
			continue
		}
		body := strings.TrimSpace(c.Body)
		if body == "" || body == "[deleted]" || body == "[removed]" {
			continue
		}
		out = append(out, Comment{
			ID:        c.ID,
			Author:    c.Author,
			Content:   body,
			CreatedAt: timestampToTime(c.Created),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// doWithRetry retries API operations with backoff and explicit support for Reddit
// rate-limit hints (Retry-After and x-ratelimit-reset).
func (f *RedditFetcher) doWithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= redditRetryAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		var apiErr *goreddit.ErrorResponse
		if errors.As(err, &apiErr) {
			if apiErr.Response.StatusCode == http.StatusForbidden {
				lastErr = fmt.Errorf("reddit returned 403 forbidden")
			}
		}

		if attempt == redditRetryAttempts {
			break
		}

		delay, retryable, reason := computeRetryDelay(err, attempt)
		if !retryable {
			break
		}

		if delay <= 0 {
			delay = redditRetryBaseDelay
		}

		f.logger.Warn(
			"Reddit request failed; retrying",
			slog.String("operation", operation),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", redditRetryAttempts),
			slog.Duration("retry_in", delay),
			slog.String("retry_reason", reason),
			slog.String("error", err.Error()),
		)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return fmt.Errorf("retry failed: %w", lastErr)
}

// computeRetryDelay classifies retryable errors and chooses an appropriate delay.
func computeRetryDelay(err error, attempt int) (time.Duration, bool, string) {
	fallback := exponentialBackoff(attempt)

	var rateLimitErr *goreddit.RateLimitError
	if errors.As(err, &rateLimitErr) {
		if d := retryAfterFromResponse(rateLimitErr.Response); d > 0 {
			return d, true, "reddit_rate_limit_retry_after"
		}
		if d := time.Until(rateLimitErr.Rate.Reset); d > 0 {
			return d, true, "reddit_rate_limit_reset"
		}
		return fallback, true, "reddit_rate_limit_fallback"
	}

	var apiErr *goreddit.ErrorResponse
	if errors.As(err, &apiErr) {
		if apiErr.Response != nil {
			switch apiErr.Response.StatusCode {
			case http.StatusTooManyRequests:
				if d := retryAfterFromHeader(apiErr.Response.Header.Get("retry-after")); d > 0 {
					return d, true, "http_429_retry_after"
				}
				if d := retryAfterFromHeader(apiErr.Response.Header.Get("x-ratelimit-reset")); d > 0 {
					return d, true, "http_429_x_ratelimit_reset"
				}
				return fallback, true, "http_429_fallback"
			case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				return fallback, true, "http_5xx"
			}
		}
	}

	// OAuth token acquisition can fail before we receive a structured response from
	// go-reddit. Treat explicit 429 token failures as retryable.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "oauth2: cannot fetch token") && strings.Contains(msg, "429 too many requests") {
		return maxDuration(15*time.Second, fallback), true, "oauth_token_429"
	}

	return 0, false, "non_retryable_error"
}

func exponentialBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := redditRetryBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= redditRetryMaxDelay {
			return redditRetryMaxDelay
		}
	}
	return delay
}

func retryAfterFromResponse(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	if d := retryAfterFromHeader(resp.Header.Get("retry-after")); d > 0 {
		return d
	}
	return retryAfterFromHeader(resp.Header.Get("x-ratelimit-reset"))
}

func retryAfterFromHeader(value string) time.Duration {
	v := strings.TrimSpace(value)
	if v == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(v); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func canonicalRedditPostURL(permalink string) string {
	if permalink == "" {
		return ""
	}
	if strings.HasPrefix(permalink, "http://") || strings.HasPrefix(permalink, "https://") {
		return permalink
	}
	if strings.HasPrefix(permalink, "/") {
		return "https://www.reddit.com" + permalink
	}
	return "https://www.reddit.com/" + permalink
}

func timestampToTime(ts *goreddit.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.UTC()
}

func extractPostURLs(post *goreddit.Post) (urls []string, images []string) {
	if post == nil {
		return nil, nil
	}

	seenURL := map[string]bool{}
	seenImage := map[string]bool{}

	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || (!strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://")) {
			return
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return
		}

		if isIgnoreURL(parsed) {
			return
		}

		normalized := parsed.String()
		if isImageURL(parsed) {
			if !seenImage[normalized] {
				seenImage[normalized] = true
				images = append(images, normalized)
			}
			return
		}
		if !seenURL[normalized] {
			seenURL[normalized] = true
			urls = append(urls, normalized)
		}
	}

	if !post.IsSelfPost && post.URL != "" {
		add(post.URL)
	}

	for _, token := range strings.FieldsFunc(post.Body, func(r rune) bool {
		switch r {
		case ' ', '\n', '\t', '\r', '(', ')', '[', ']', '{', '}', '<', '>', '"', '\'':
			return true
		default:
			return false
		}
	}) {
		add(token)
	}

	return urls, images
}

func isImageURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Host)
	switch host {
	case "i.redd.it", "i.imgur.com":
		return true
	}
	path := strings.ToLower(u.Path)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func isIgnoreURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Host)

	switch host {
	case "preview.redd.it", "localhost", "discord.gg":
		return true
	}

	// Ignore user profile links
	if (host == "www.reddit.com" || host == "old.reddit.com") && (strings.HasPrefix(u.Path, "/user/") || strings.HasPrefix(u.Path, "/u/")) {
		return true
	}

	return false
}
