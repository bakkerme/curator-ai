package summary

import (
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
	"github.com/bakkerme/curator-ai/internal/processors/llmutil"
)

var RETRIES = 3

type PostLLMProcessor struct {
	name           string
	config         config.LLMSummary
	client         llm.Client
	defaultModel   string
	systemTemplate *template.Template
	template       *template.Template
}

func NewPostLLMProcessor(cfg *config.LLMSummary, client llm.Client, defaultModel string) (*PostLLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("summary config is required")
	}

	systemTmpl, tmpl, err := llmutil.ParseSystemAndPromptTemplates(cfg.Name, cfg.SystemTemplate, cfg.PromptTemplate)
	if err != nil {
		return nil, err
	}

	return &PostLLMProcessor{
		name:           cfg.Name,
		config:         *cfg,
		client:         client,
		defaultModel:   defaultModel,
		systemTemplate: systemTmpl,
		template:       tmpl,
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
		data := struct {
			*core.PostBlock
			Params map[string]interface{}
		}{
			PostBlock: block,
			Params:    p.config.Params,
		}

		systemPrompt, err := llmutil.ExecuteTemplate(p.systemTemplate, data)
		if err != nil {
			return nil, err
		}

		userPrompt, err := llmutil.ExecuteTemplate(p.template, data)
		if err != nil {
			return nil, err
		}

		model := llmutil.ModelOrDefault(p.config.Model, p.defaultModel)
		response, err := llmutil.ChatSystemUserWithRetries(ctx, p.client, model, systemPrompt, userPrompt, RETRIES, nil)
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
