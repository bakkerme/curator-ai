package arxiv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/dedupe"
	"github.com/bakkerme/curator-ai/internal/sources"
	"github.com/bakkerme/curator-ai/internal/sources/reader"
)

// ArxivProcessor fetches papers from arXiv and emits PostBlocks with chunked content.
// Full text is loaded from the paper PDF through the configured reader.
type ArxivProcessor struct {
	name    string
	config  config.ArxivSource
	fetcher Fetcher
	reader  reader.Reader
	store   dedupe.SeenStore
	logger  *slog.Logger
}

// NewArxivProcessor wires a new arXiv source processor.
func NewArxivProcessor(cfg *config.ArxivSource, fetcher Fetcher, r reader.Reader, store dedupe.SeenStore, logger *slog.Logger) (*ArxivProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("arxiv config is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ArxivProcessor{
		name:    "arxiv",
		config:  *cfg,
		fetcher: fetcher,
		reader:  r,
		store:   store,
		logger:  logger,
	}, nil
}

func (p *ArxivProcessor) Name() string {
	return p.name
}

func (p *ArxivProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *ArxivProcessor) Validate() error {
	if strings.TrimSpace(p.config.Query) == "" && len(p.config.Categories) == 0 {
		return fmt.Errorf("arxiv query or categories are required")
	}
	if p.fetcher == nil {
		return fmt.Errorf("arxiv fetcher is required")
	}
	if p.reader == nil {
		return fmt.Errorf("arxiv reader is required")
	}
	return nil
}

func (p *ArxivProcessor) Fetch(ctx context.Context) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	logger := core.LoggerFromContext(ctx).With("stage", "source", "processor", p.name)
	options := SearchOptions{
		Query:      p.config.Query,
		Categories: p.config.Categories,
		MaxResults: p.config.MaxResults,
		SortBy:     p.config.SortBy,
		SortOrder:  p.config.SortOrder,
		DateFrom:   p.config.DateFrom,
		DateTo:     p.config.DateTo,
	}
	logger.Info("Fetching papers from arXiv", "query", options.Query, "categories", options.Categories)
	papers, err := p.fetcher.Search(ctx, options)
	if err != nil {
		return nil, err
	}

	includeAbstractInChunks := true
	if p.config.IncludeAbstractInChunks != nil {
		includeAbstractInChunks = *p.config.IncludeAbstractInChunks
	}
	abstractOnly := p.config.AbstractOnly != nil && *p.config.AbstractOnly
	chunking := defaultArxivChunkingConfig(p.config.Chunking)

	blocks := make([]*core.PostBlock, 0, len(papers))
	for _, paper := range papers {
		if paper.ID == "" {
			logger.Warn("Skipping arXiv paper without ID", "title", paper.Title)
			continue
		}
		if p.store != nil {
			seen, err := p.store.HasSeen(ctx, paper.ID)
			if err != nil {
				logger.Warn("Failed to check dedupe store", "paper_id", paper.ID, "error", err)
			} else if seen {
				logger.Info("Skipping already seen paper", "paper_id", paper.ID)
				continue
			}
		}

		var chunks []core.ContentChunk
		var content string
		var errors []core.ProcessError
		if abstractOnly {
			// Abstract-only mode ensures downstream processors receive only abstract text.
			content = paper.Abstract
			chunks = chunkArxivContent(content, paper.Abstract, false, chunking)
			logger.Info("Using abstract-only mode for arXiv paper", "paper_id", paper.ID)
		} else {
			content, errors = p.fetchPaperContent(ctx, logger, paper)
			if strings.TrimSpace(content) == "" {
				content = paper.Abstract
				chunks = chunkArxivContent(content, paper.Abstract, false, chunking)
				logger.Error("Failed to fetch full text content; using abstract only", "paper_id", paper.ID)
			} else {
				chunks = chunkArxivContent(content, paper.Abstract, includeAbstractInChunks, chunking)
			}
		}

		block := &core.PostBlock{
			ID:          paper.ID,
			URL:         paper.AbsURL,
			Title:       paper.Title,
			Content:     content,
			Author:      strings.Join(paper.Authors, ", "),
			CreatedAt:   paper.PublishedAt,
			SummaryPlan: sources.SummaryPlanFromConfig(p.config.SummaryPlan),
			Chunks:      chunks,
			ProcessedAt: time.Now().UTC(),
		}
		if len(errors) > 0 {
			block.Errors = append(block.Errors, errors...)
		}
		blocks = append(blocks, block)

		if p.store != nil {
			if err := p.store.MarkSeen(ctx, paper.ID); err != nil {
				logger.Warn("Failed to mark paper as seen", "paper_id", paper.ID, "error", err)
			}
		}
	}

	return blocks, nil
}

func (p *ArxivProcessor) fetchPaperContent(ctx context.Context, logger *slog.Logger, paper Paper) (string, []core.ProcessError) {
	var errors []core.ProcessError
	if paper.PDFURL == "" {
		errors = append(errors, core.ProcessError{
			ProcessorName: p.name,
			Stage:         "source",
			Error:         "arxiv pdf url missing; unable to fetch full text",
			OccurredAt:    time.Now().UTC(),
		})
		return "", errors
	}

	logger.Info("Fetching arXiv PDF via reader", "paper_id", paper.ID, "url", paper.PDFURL)
	content, err := p.reader.Read(ctx, paper.PDFURL)
	if err != nil || strings.TrimSpace(content) == "" {
		msg := "arxiv pdf fetch failed"
		if err != nil {
			msg = fmt.Sprintf("arxiv pdf fetch failed: %v", err)
		}
		errors = append(errors, core.ProcessError{
			ProcessorName: p.name,
			Stage:         "source",
			Error:         msg,
			OccurredAt:    time.Now().UTC(),
		})
		return "", errors
	}
	return content, nil
}
