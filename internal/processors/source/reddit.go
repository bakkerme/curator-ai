package source

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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
		if p.config.IncludeComments && len(item.Comments) > 0 {
			block.Comments = make([]core.CommentBlock, 0, len(item.Comments))
			for _, c := range item.Comments {
				cb := core.CommentBlock{
					ID:        c.ID,
					Author:    c.Author,
					Content:   c.Content,
					CreatedAt: c.CreatedAt,
				}
				if p.config.IncludeWeb || p.config.IncludeImages {
					urls, images := extractURLs(c.Content)
					if p.config.IncludeWeb {
						cb.URLs = make([]core.WebBlock, 0, len(urls))
						for _, u := range urls {
							cb.URLs = append(cb.URLs, core.WebBlock{URL: u})
						}
					}
					if p.config.IncludeImages {
						cb.Images = make([]core.ImageBlock, 0, len(images))
						for _, u := range images {
							cb.Images = append(cb.Images, core.ImageBlock{URL: u})
						}
					}
				}
				block.Comments = append(block.Comments, cb)
			}
		}
		if p.config.IncludeWeb && len(item.WebURLs) > 0 {
			block.WebBlocks = make([]core.WebBlock, 0, len(item.WebURLs))
			for _, u := range item.WebURLs {
				block.WebBlocks = append(block.WebBlocks, core.WebBlock{URL: u})
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
	}
	return blocks, nil
}

func extractURLs(text string) (urls []string, images []string) {
	seenURL := map[string]bool{}
	seenImage := map[string]bool{}

	fields := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case ' ', '\n', '\t', '\r', '(', ')', '[', ']', '{', '}', '<', '>', '"', '\'':
			return true
		default:
			return false
		}
	})

	for _, f := range fields {
		if !strings.HasPrefix(f, "http://") && !strings.HasPrefix(f, "https://") {
			continue
		}
		parsed, err := url.Parse(f)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		normalized := parsed.String()
		if isImageURL(parsed) {
			if !seenImage[normalized] {
				seenImage[normalized] = true
				images = append(images, normalized)
			}
			continue
		}
		if !seenURL[normalized] {
			seenURL[normalized] = true
			urls = append(urls, normalized)
		}
	}

	return urls, images
}

func isImageURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Host)
	switch host {
	case "i.redd.it", "preview.redd.it", "i.imgur.com":
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
