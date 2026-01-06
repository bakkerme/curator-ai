package impl

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
	"github.com/mmcdole/gofeed"
)

type Fetcher struct {
	client *http.Client
	parser *gofeed.Parser
}

func NewFetcher(timeout time.Duration, userAgent string) *Fetcher {
	client := &http.Client{Timeout: timeout}
	parser := gofeed.NewParser()
	parser.Client = client
	parser.UserAgent = userAgent
	return &Fetcher{client: client, parser: parser}
}

func (f *Fetcher) Fetch(ctx context.Context, feedURL string, options rss.FetchOptions) ([]rss.Item, error) {
	var feed *gofeed.Feed
	err := retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		parsed, err := f.parser.ParseURLWithContext(feedURL, ctx)
		if err != nil {
			return err
		}
		feed = parsed
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	limit := options.Limit
	if limit <= 0 {
		limit = len(feed.Items)
	}

	items := make([]rss.Item, 0, limit)
	for _, entry := range feed.Items {
		if len(items) >= limit {
			break
		}
		item := rss.Item{
			ID:          entry.GUID,
			Title:       entry.Title,
			Link:        entry.Link,
			Description: entry.Description,
			Content:     entry.Content,
			Author:      "",
		}
		if entry.Author != nil {
			item.Author = entry.Author.Name
		}
		if entry.PublishedParsed != nil {
			item.PublishedAt = *entry.PublishedParsed
		} else if entry.UpdatedParsed != nil {
			item.PublishedAt = *entry.UpdatedParsed
		} else {
			item.PublishedAt = time.Now().UTC()
		}
		items = append(items, item)
	}

	return items, nil
}
