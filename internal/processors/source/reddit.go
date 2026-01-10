package source

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/dedupe"
	"github.com/bakkerme/curator-ai/internal/sources/jina"
	"github.com/bakkerme/curator-ai/internal/sources/reddit"
)

type RedditProcessor struct {
	name    string
	config  config.RedditSource
	fetcher reddit.Fetcher
	reader  jina.Reader
	store   dedupe.SeenStore
	logger  *slog.Logger
}

func NewRedditProcessor(cfg *config.RedditSource, fetcher reddit.Fetcher, reader jina.Reader, store dedupe.SeenStore, logger *slog.Logger) (*RedditProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("reddit config is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &RedditProcessor{
		name:    "reddit",
		config:  *cfg,
		fetcher: fetcher,
		reader:  reader,
		store:   store,
		logger:  logger,
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
	p.logger.Info("Fetching posts from Reddit", slog.Int("subreddits", len(p.config.Subreddits)))
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
		if p.store != nil {
			seen, err := p.store.HasSeen(ctx, item.ID)
			if err != nil {
				p.logger.Warn("Failed to check dedupe store", slog.String("post_id", item.ID), slog.String("error", err.Error()))
			} else if seen {
				p.logger.Info("Skipping already seen post", slog.String("post_id", item.ID))
				continue
			}
		}

		block := &core.PostBlock{
			ID:        item.ID,
			URL:       item.URL,
			Title:     item.Title,
			Content:   item.Content,
			Author:    item.Author,
			CreatedAt: item.CreatedAt,
		}

		if p.config.IncludeComments && len(item.Comments) > 0 {
			block.Comments = make([]core.CommentBlock, 0, len(item.Comments))
			for _, c := range item.Comments {
				cb := core.CommentBlock{
					ID:        c.ID,
					Author:    c.Author,
					Content:   c.Content,
					CreatedAt: c.CreatedAt,
				}

				block.Comments = append(block.Comments, cb)
			}
		}

		if p.config.IncludeWeb && len(item.WebURLs) > 0 {
			p.logger.Info("Fetching web URLs via Jina", slog.String("post_id", item.ID), slog.Int("urls", len(item.WebURLs)))
			block.WebBlocks = make([]core.WebBlock, 0, len(item.WebURLs))
			for _, u := range item.WebURLs {
				wb := core.WebBlock{URL: u}
				started := time.Now()
				p.logger.Info("Fetching URL via Jina", slog.String("post_id", item.ID), slog.String("url", u))

				page, err := p.reader.Read(ctx, u, jina.ReadOptions{})
				if err != nil {
					p.logger.Warn("Failed to fetch URL via Jina", slog.String("post_id", item.ID), slog.String("url", u), slog.Duration("elapsed", time.Since(started)), slog.String("error", err.Error()))
					block.Errors = append(block.Errors, core.ProcessError{
						ProcessorName: p.name,
						Stage:         "source",
						Error:         fmt.Sprintf("jina fetch %s: %v", u, err),
						OccurredAt:    time.Now().UTC(),
					})
				} else {
					p.logger.Info("Fetched URL via Jina", slog.String("post_id", item.ID), slog.String("url", u), slog.Duration("elapsed", time.Since(started)))
					wb.WasFetched = true
					wb.Page = page
				}

				block.WebBlocks = append(block.WebBlocks, wb)
			}
		}

		if p.config.IncludeImages && len(item.ImageURLs) > 0 {
			block.ImageBlocks = make([]core.ImageBlock, 0, len(item.ImageURLs))
			for _, u := range item.ImageURLs {
				block.ImageBlocks = append(block.ImageBlocks, core.ImageBlock{URL: u})
			}
		}
		block.ProcessedAt = time.Now().UTC()
		blocks = append(blocks, block)

		if p.store != nil {
			if err := p.store.MarkSeen(ctx, item.ID); err != nil {
				p.logger.Warn("Failed to mark post as seen", slog.String("post_id", item.ID), slog.String("error", err.Error()))
			}
		}
	}
	return blocks, nil
}
