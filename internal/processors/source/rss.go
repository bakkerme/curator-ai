package source

import (
	"context"
	"fmt"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/sources/rss"
)

type RSSProcessor struct {
	name    string
	config  config.RSSSource
	fetcher rss.Fetcher
}

func NewRSSProcessor(cfg *config.RSSSource, fetcher rss.Fetcher) (*RSSProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rss config is required")
	}
	return &RSSProcessor{
		name:    "rss",
		config:  *cfg,
		fetcher: fetcher,
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

	includeContent := true
	if p.config.IncludeContent != nil {
		includeContent = *p.config.IncludeContent
	}

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
			id := item.ID
			if id == "" {
				id = item.Link
			}
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true

			content := item.Description
			if includeContent && item.Content != "" {
				content = item.Content
			}
			if content == "" {
				content = item.Description
			}

			block := &core.PostBlock{
				ID:        id,
				URL:       item.Link,
				Title:     item.Title,
				Content:   content,
				Author:    item.Author,
				CreatedAt: item.PublishedAt,
			}
			block.ProcessedAt = time.Now().UTC()
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}
