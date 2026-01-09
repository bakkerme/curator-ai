package reddit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
	goreddit "github.com/vartanbeno/go-reddit/v2/reddit"
)

type RedditFetcher struct {
	client  *goreddit.Client
	initErr error
	logger  *slog.Logger
}

func NewFetcher(logger *slog.Logger, timeout time.Duration, userAgent, clientID, clientSecret, username, password string) Fetcher {
	if userAgent == "" {
		userAgent = "curator-ai/0.1"
	}

	httpClient := &http.Client{Timeout: timeout}
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

	return &RedditFetcher{client: client, initErr: err, logger: logger}
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

	return items, nil
}

func (f RedditFetcher) fetchPosts(ctx context.Context, subreddit, sort string, limit int, timeFilter string) ([]*goreddit.Post, error) {
	var (
		posts []*goreddit.Post
		resp  *goreddit.Response
	)
	err := retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		var err error
		switch strings.ToLower(sort) {
		case "hot":
			posts, resp, err = f.client.Subreddit.HotPosts(ctx, subreddit, &goreddit.ListOptions{Limit: limit})
		case "new":
			posts, resp, err = f.client.Subreddit.NewPosts(ctx, subreddit, &goreddit.ListOptions{Limit: limit})
		case "rising":
			posts, resp, err = f.client.Subreddit.RisingPosts(ctx, subreddit, &goreddit.ListOptions{Limit: limit})
		case "top":
			posts, resp, err = f.client.Subreddit.TopPosts(ctx, subreddit, &goreddit.ListPostOptions{
				ListOptions: goreddit.ListOptions{Limit: limit},
				Time:        timeFilter,
			})
		case "controversial":
			posts, resp, err = f.client.Subreddit.ControversialPosts(ctx, subreddit, &goreddit.ListPostOptions{
				ListOptions: goreddit.ListOptions{Limit: limit},
				Time:        timeFilter,
			})
		default:
			return fmt.Errorf("unsupported reddit sort: %q", sort)
		}
		if err != nil {
			if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
				return fmt.Errorf("reddit transient error: %w", err)
			}
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
		pc   *goreddit.PostAndComments
		resp *goreddit.Response
	)
	err := retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		var err error
		pc, resp, err = f.client.Post.Get(ctx, postID)
		if err != nil {
			if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
				return fmt.Errorf("reddit transient error: %w", err)
			}
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
	return ts.Time.UTC()
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
	return false
}
