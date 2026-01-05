package summary

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type PostLLMProcessor struct {
	name         string
	config       config.LLMSummary
	client       llm.Client
	defaultModel string
	template     *template.Template
}

func NewPostLLMProcessor(cfg *config.LLMSummary, client llm.Client, defaultModel string) (*PostLLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("summary config is required")
	}
	tmpl, err := template.New(cfg.Name).Parse(cfg.PromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template: %w", err)
	}
	return &PostLLMProcessor{
		name:         cfg.Name,
		config:       *cfg,
		client:       client,
		defaultModel: defaultModel,
		template:     tmpl,
	}, nil
}

func (p *PostLLMProcessor) Name() string {
	return p.name
}

func (p *PostLLMProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *PostLLMProcessor) Validate() error {
	if p.client == nil {
		return fmt.Errorf("llm client is required")
	}
	if p.config.Context != "post" {
		return fmt.Errorf("summary context must be post")
	}
	return nil
}

func (p *PostLLMProcessor) Summarize(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	for _, block := range blocks {
		prompt, err := executeTemplate(p.template, block)
		if err != nil {
			return nil, err
		}
		model := p.config.Model
		if model == "" {
			model = p.defaultModel
		}
		response, err := p.client.ChatCompletion(ctx, llm.ChatRequest{
			Model: model,
			Messages: []llm.Message{
				{Role: llm.RoleUser, Content: prompt},
			},
		})
		if err != nil {
			return nil, err
		}
		block.Summary = &core.SummaryResult{
			ProcessorName: p.name,
			Summary:       response.Content,
			ProcessedAt:   time.Now().UTC(),
		}
	}
	return blocks, nil
}

func executeTemplate(tmpl *template.Template, data interface{}) (string, error) {
	builder := &strings.Builder{}
	if err := tmpl.Execute(builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}
