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

type RunLLMProcessor struct {
	name         string
	config       config.LLMSummary
	client       llm.Client
	defaultModel string
	template     *template.Template
}

func NewRunLLMProcessor(cfg *config.LLMSummary, client llm.Client, defaultModel string) (*RunLLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("run summary config is required")
	}
	tmpl, err := template.New(cfg.Name).Parse(cfg.PromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template: %w", err)
	}
	return &RunLLMProcessor{
		name:         cfg.Name,
		config:       *cfg,
		client:       client,
		defaultModel: defaultModel,
		template:     tmpl,
	}, nil
}

func (p *RunLLMProcessor) Name() string {
	return p.name
}

func (p *RunLLMProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *RunLLMProcessor) Validate() error {
	if p.client == nil {
		return fmt.Errorf("llm client is required")
	}
	if p.config.Context != "flow" {
		return fmt.Errorf("run summary context must be flow")
	}
	return nil
}

func (p *RunLLMProcessor) SummarizeRun(ctx context.Context, blocks []*core.PostBlock) (*core.RunSummary, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	prompt, err := executeRunTemplate(p.template, blocks)
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
	return &core.RunSummary{
		ProcessorName: p.name,
		Summary:       response.Content,
		PostCount:     len(blocks),
		ProcessedAt:   time.Now().UTC(),
	}, nil
}

func executeRunTemplate(tmpl *template.Template, data interface{}) (string, error) {
	builder := &strings.Builder{}
	if err := tmpl.Execute(builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}
