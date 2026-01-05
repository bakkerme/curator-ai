package source

import (
	"context"
	"fmt"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
)

type RedditProcessor struct {
	name    string
	config  config.RedditSource
	fetcher reddit.Fetcher
}

func NewRedditProcessor(cfg *config.RedditSource, fetcher reddit.Fetcher) (*RedditProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("reddit config is required")
	}
	return &RedditProcessor{
		name:    "reddit",
		config:  *cfg,
		fetcher: fetcher,
	}, nil
}

func (p *RedditProcessor) Name() string {
	return p.name
}

func (p *RedditProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *RedditProcessor) Validate() error {
	if len(p.config.Subreddits) == 0 {
		return fmt.Errorf("at least one subreddit is required")
	}
	if p.fetcher == nil {
		return fmt.Errorf("reddit fetcher is required")
	}
	return nil
}

func (p *RedditProcessor) Fetch(ctx context.Context) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	items, err := p.fetcher.Fetch(ctx, reddit.Config{
		Subreddits:      p.config.Subreddits,
		Limit:           p.config.Limit,
		Sort:            p.config.Sort,
		TimeFilter:      p.config.TimeFilter,
		IncludeComments: p.config.IncludeComments,
		IncludeWeb:      p.config.IncludeWeb,
		IncludeImages:   p.config.IncludeImages,
		MinScore:        p.config.MinScore,
		UserAgent:       "",
	})
	if err != nil {
		return nil, err
	}

	blocks := make([]*core.PostBlock, 0, len(items))
	for _, item := range items {
		block := &core.PostBlock{
			ID:        item.ID,
			URL:       item.URL,
			Title:     item.Title,
			Content:   item.Content,
			Author:    item.Author,
			CreatedAt: item.CreatedAt,
		}
		block.ProcessedAt = time.Now().UTC()
		blocks = append(blocks, block)
	}
	return blocks, nil
}
