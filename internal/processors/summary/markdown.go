package summary

import (
	"context"
	"fmt"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	rendermarkdown "github.com/bakkerme/curator-ai/internal/render/markdown"
)

type PostMarkdownProcessor struct {
	name   string
	config config.MarkdownSummary
}

func NewPostMarkdownProcessor(cfg *config.MarkdownSummary) (*PostMarkdownProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("markdown summary config is required")
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("markdown summary name is required")
	}
	return &PostMarkdownProcessor{
		name:   cfg.Name,
		config: *cfg,
	}, nil
}

func (p *PostMarkdownProcessor) Name() string {
	return p.name
}

func (p *PostMarkdownProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *PostMarkdownProcessor) Validate() error {
	if p.config.Context != "post" {
		return fmt.Errorf("markdown summary context must be post")
	}
	return nil
}

func (p *PostMarkdownProcessor) Summarize(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	for _, block := range blocks {
		if block.Summary == nil {
			return nil, fmt.Errorf("markdown summary requires existing post summary")
		}
		html, err := rendermarkdown.Render(block.Summary.Summary)
		if err != nil {
			return nil, err
		}
		block.Summary.HTML = html
		block.Summary.ProcessorName = p.name
		block.Summary.ProcessedAt = time.Now().UTC()
	}
	return blocks, nil
}

type RunMarkdownProcessor struct {
	name   string
	config config.MarkdownSummary
}

func NewRunMarkdownProcessor(cfg *config.MarkdownSummary) (*RunMarkdownProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("run markdown summary config is required")
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("run markdown summary name is required")
	}
	return &RunMarkdownProcessor{
		name:   cfg.Name,
		config: *cfg,
	}, nil
}

func (p *RunMarkdownProcessor) Name() string {
	return p.name
}

func (p *RunMarkdownProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *RunMarkdownProcessor) Validate() error {
	if p.config.Context != "flow" {
		return fmt.Errorf("run markdown summary context must be flow")
	}
	return nil
}

func (p *RunMarkdownProcessor) SummarizeRun(ctx context.Context, blocks []*core.PostBlock, current *core.RunSummary) (*core.RunSummary, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("markdown run summary requires an existing run summary")
	}
	html, err := rendermarkdown.Render(current.Summary)
	if err != nil {
		return nil, err
	}
	updated := *current
	updated.HTML = html
	updated.ProcessorName = p.name
	updated.ProcessedAt = time.Now().UTC()
	return &updated, nil
}
