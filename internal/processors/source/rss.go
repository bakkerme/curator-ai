package source

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/dedupe"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
)

type RSSProcessor struct {
	name    string
	config  config.RSSSource
	fetcher rss.Fetcher
	store   dedupe.SeenStore
}

func NewRSSProcessor(cfg *config.RSSSource, fetcher rss.Fetcher, store dedupe.SeenStore) (*RSSProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rss config is required")
	}
	return &RSSProcessor{
		name:    "rss",
		config:  *cfg,
		fetcher: fetcher,
		store:   store,
	}, nil
}

func (p *RSSProcessor) Name() string {
	return p.name
}

func (p *RSSProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *RSSProcessor) Validate() error {
	if len(p.config.Feeds) == 0 {
		return fmt.Errorf("at least one rss feed is required")
	}
	if p.fetcher == nil {
		return fmt.Errorf("rss fetcher is required")
	}
	return nil
}

func (p *RSSProcessor) Fetch(ctx context.Context) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	logger := core.LoggerFromContext(ctx).With("stage", "source", "processor", p.name)

	includeContent := true
	if p.config.IncludeContent != nil {
		includeContent = *p.config.IncludeContent
	}

	convertSourceToMarkdown := p.config.ConvertSourceToMarkdown

	blocks := []*core.PostBlock{}
	seen := map[string]bool{}

	options := rss.FetchOptions{
		Limit:     p.config.Limit,
		UserAgent: p.config.UserAgent,
	}

	for _, feedURL := range p.config.Feeds {
		items, err := p.fetcher.Fetch(ctx, feedURL, options)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			postLogger := logger.With("feed_url", feedURL)

			id := item.ID
			if id == "" {
				id = item.Link
			}
			if id == "" || seen[id] {
				continue
			}
			if p.store != nil {
				alreadySeen, err := p.store.HasSeen(ctx, id)
				if err != nil {
					postLogger.Warn("failed to check dedupe store", "post_id", id, "error", err)
				} else if alreadySeen {
					postLogger.Info("skipping already seen post", "post_id", id)
					continue
				}
			}
			seen[id] = true

			content := item.Description
			if includeContent && item.Content != "" {
				content = item.Content
			}
			if content == "" {
				content = item.Description
			}

			var procErrors []core.ProcessError

			// Extract embedded data: images into ImageBlocks and replace the <img> src
			// with a small placeholder URL so the base64 doesn't burn tokens downstream.
			placeholderBase := "curator-image://post/" + url.PathEscape(id)
			scrubbed, images, err := rss.ExtractDataURIImagesFromHTML(content, placeholderBase)
			if err == nil {
				content = scrubbed
			}
			if err != nil {
				postLogger.Warn(
					"failed to extract data URI images from HTML",
					"post_id", id,
					"post_url", item.Link,
					"error", err,
				)
				procErrors = append(procErrors, core.ProcessError{
					ProcessorName: p.name,
					Stage:         "source",
					Error:         err.Error(),
					OccurredAt:    time.Now().UTC(),
				})
			}

			// Convert to markdown if needed
			if convertSourceToMarkdown {
				mdContent, err := rss.ConvertHTMLToMarkdown(content)
				if err == nil && mdContent != "" {
					content = mdContent
				}
				if err != nil {
					postLogger.Warn(
						"failed to convert HTML to Markdown",
						"post_id", id,
						"post_url", item.Link,
						"error", err,
					)
					procErrors = append(procErrors, core.ProcessError{
						ProcessorName: p.name,
						Stage:         "source",
						Error:         err.Error(),
						OccurredAt:    time.Now().UTC(),
					})
				}
			}

			block := &core.PostBlock{
				ID:        id,
				URL:       item.Link,
				Title:     item.Title,
				Content:   content,
				Author:    item.Author,
				CreatedAt: item.PublishedAt,
			}
			if len(images) > 0 {
				block.ImageBlocks = append(block.ImageBlocks, images...)
			}
			if len(procErrors) > 0 {
				block.Errors = append(block.Errors, procErrors...)
			}
			block.ProcessedAt = time.Now().UTC()
			blocks = append(blocks, block)

			if p.store != nil {
				if err := p.store.MarkSeen(ctx, id); err != nil {
					postLogger.Warn("failed to mark post as seen", "post_id", id, "error", err)
				}
			}
		}
	}

	return blocks, nil
}
