package scrape

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/dedupe"
	"github.com/bakkerme/curator-ai/internal/htmlutil"
	"github.com/bakkerme/curator-ai/internal/sources"
)

// ScrapeProcessor discovers post URLs from index pages and extracts content from post pages.
type ScrapeProcessor struct {
	name       string
	config     config.ScrapeSource
	fetcher    Fetcher
	store      dedupe.SeenStore
	logger     *slog.Logger
	fetchDelay time.Duration
}

func NewScrapeProcessor(cfg *config.ScrapeSource, fetcher Fetcher, store dedupe.SeenStore, logger *slog.Logger) (*ScrapeProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("scrape config is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	var fetchDelay time.Duration
	if d := strings.TrimSpace(cfg.Request.FetchDelay); d != "" {
		var err error
		fetchDelay, err = config.ParseDurationExtended(d)
		if err != nil {
			return nil, fmt.Errorf("invalid fetch_delay %q: %w", d, err)
		}
	}
	return &ScrapeProcessor{name: "scrape", config: *cfg, fetcher: fetcher, store: store, logger: logger, fetchDelay: fetchDelay}, nil
}

func (p *ScrapeProcessor) Name() string                                  { return p.name }
func (p *ScrapeProcessor) Configure(config map[string]interface{}) error { return nil }

func (p *ScrapeProcessor) Validate() error {
	if p.fetcher == nil {
		return fmt.Errorf("scrape fetcher is required")
	}
	if strings.TrimSpace(p.config.URL) == "" {
		return fmt.Errorf("scrape url is required")
	}
	if strings.TrimSpace(p.config.Discovery.ItemSelector) == "" {
		return fmt.Errorf("scrape discovery.item_selector is required")
	}
	if strings.TrimSpace(p.config.Extraction.ContentSelector) == "" {
		return fmt.Errorf("scrape extraction.content_selector is required")
	}
	return nil
}

func (p *ScrapeProcessor) Fetch(ctx context.Context) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	logger := core.LoggerFromContext(ctx).With("stage", "source", "processor", p.name)

	cutoff, hasLookback, err := parseLookbackWindow(p.config.Lookback)
	if err != nil {
		return nil, err
	}
	options := FetchOptions{UserAgent: p.config.Request.UserAgent}
	maxPages := p.config.Discovery.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	postLimit := p.config.PostLimit

	blocks := make([]*core.PostBlock, 0)
	discovered := map[string]struct{}{}
	stopReason := ""
	fetchCount := 0

	nextPageURL := strings.TrimSpace(p.config.URL)
	for page := 1; page <= maxPages && nextPageURL != ""; page++ {
		if postLimit > 0 && len(blocks) >= postLimit {
			stopReason = "post_limit"
			break
		}

		if err := p.delayIfNeeded(ctx, &fetchCount); err != nil {
			return nil, err
		}
		indexHTML, err := p.fetcher.Fetch(ctx, nextPageURL, options)
		if err != nil {
			return nil, fmt.Errorf("fetch index page %s: %w", nextPageURL, err)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(indexHTML))
		if err != nil {
			return nil, fmt.Errorf("parse index page %s: %w", nextPageURL, err)
		}

		links := p.discoverLinks(doc, nextPageURL)
		for _, postURL := range links {
			if _, ok := discovered[postURL]; ok {
				continue
			}
			discovered[postURL] = struct{}{}

			if p.store != nil {
				seen, err := p.store.HasSeen(ctx, postURL)
				if err != nil {
					logger.Warn("failed to check dedupe store", "post_url", postURL, "error", err)
				} else if seen {
					continue
				}
			}

			if err := p.delayIfNeeded(ctx, &fetchCount); err != nil {
				return nil, err
			}
			block, skip, err := p.extractPost(ctx, postURL, page, cutoff, hasLookback, options)
			if err != nil {
				logger.Warn("failed to extract scraped post", "post_url", postURL, "error", err)
				continue
			}
			if skip || block == nil {
				continue
			}

			blocks = append(blocks, block)
			if p.store != nil {
				if err := p.store.MarkSeen(ctx, postURL); err != nil {
					logger.Warn("failed to mark scraped post as seen", "post_url", postURL, "error", err)
				}
			}
			if postLimit > 0 && len(blocks) >= postLimit {
				stopReason = "post_limit"
				break
			}
		}

		if stopReason != "" {
			break
		}

		next := strings.TrimSpace(p.config.Discovery.NextPageSelector)
		if next == "" {
			nextPageURL = ""
		} else {
			href, _ := doc.Find(next).First().Attr("href")
			nextPageURL = resolveURL(nextPageURL, href)
		}
	}
	if stopReason == "" && maxPages > 0 {
		stopReason = "max_pages"
	}

	for _, block := range blocks {
		if block.Metadata == nil {
			block.Metadata = map[string]string{}
		}
		if stopReason != "" {
			block.Metadata["source_stop_reason"] = stopReason
		}
		if hasLookback {
			block.Metadata["source_lookback"] = p.config.Lookback
			block.Metadata["source_cutoff"] = cutoff.UTC().Format(time.RFC3339)
		}
	}

	logger.Info("scrape source completed", "posts", len(blocks), "stop_reason", stopReason, "lookback_enabled", hasLookback)
	return blocks, nil
}

// delayIfNeeded sleeps for the configured fetch_delay between HTTP requests.
// The first request (fetchCount == 0) is never delayed.
func (p *ScrapeProcessor) delayIfNeeded(ctx context.Context, fetchCount *int) error {
	if p.fetchDelay > 0 && *fetchCount > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(p.fetchDelay):
		}
	}
	*fetchCount++
	return nil
}

func (p *ScrapeProcessor) discoverLinks(doc *goquery.Document, baseURL string) []string {
	attr := strings.TrimSpace(p.config.Discovery.LinkAttr)
	if attr == "" {
		attr = "href"
	}
	urls := make([]string, 0)
	doc.Find(p.config.Discovery.ItemSelector).Each(func(_ int, sel *goquery.Selection) {
		if href, ok := sel.Attr(attr); ok {
			if resolved := resolveURL(baseURL, href); resolved != "" {
				urls = append(urls, resolved)
			}
		}
	})
	return urls
}

func (p *ScrapeProcessor) extractPost(ctx context.Context, postURL string, page int, cutoff time.Time, hasLookback bool, options FetchOptions) (*core.PostBlock, bool, error) {
	html, err := p.fetcher.Fetch(ctx, postURL, options)
	if err != nil {
		return nil, false, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, false, err
	}

	for _, remove := range p.config.Extraction.RemoveSelectors {
		doc.Find(remove).Remove()
	}

	title := extractValue(doc, p.config.Extraction.TitleSelector, p.config.Extraction.TitleAttr)
	author := extractValue(doc, p.config.Extraction.AuthorSelector, p.config.Extraction.AuthorAttr)
	dateRaw := extractValue(doc, p.config.Extraction.DateSelector, p.config.Extraction.DateAttr)
	contentSel := doc.Find(p.config.Extraction.ContentSelector).First()
	contentHTML, err := contentSel.Html()
	if err != nil {
		return nil, false, fmt.Errorf("extract content html: %w", err)
	}
	content := strings.TrimSpace(contentHTML)

	var webBlocks []core.WebBlock
	if p.config.Markdown.Enabled {
		webBlocks = []core.WebBlock{{
			URL:        postURL,
			WasFetched: true,
			Page:       content,
		}}
		md, err := htmlutil.ConvertHTMLToMarkdown(content)
		if err != nil {
			return nil, false, err
		}
		content = md
	}

	createdAt, ok := parseTimeFlexible(dateRaw)
	if hasLookback && ok && createdAt.Before(cutoff) {
		return nil, true, nil
	}
	if !ok {
		createdAt = time.Now().UTC()
	}
	if strings.TrimSpace(title) == "" {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}
	if strings.TrimSpace(title) == "" {
		title = postURL
	}

	return &core.PostBlock{
		ID:          postURL,
		URL:         postURL,
		Title:       title,
		Author:      author,
		Content:     content,
		WebBlocks:   webBlocks,
		CreatedAt:   createdAt,
		ProcessedAt: time.Now().UTC(),
		SummaryPlan: sources.SummaryPlanFromConfig(p.config.SummaryPlan),
		Metadata: map[string]string{
			"source_processor":  p.name,
			"source_page_index": fmt.Sprintf("%d", page),
			"source_date_raw":   dateRaw,
		},
	}, false, nil
}
