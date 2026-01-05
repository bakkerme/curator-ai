package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type LLMProcessor struct {
	name         string
	config       config.LLMQuality
	client       llm.Client
	defaultModel string
	template     *template.Template
}

type qualityResponse struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

func NewLLMProcessor(cfg *config.LLMQuality, client llm.Client, defaultModel string) (*LLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm quality config is required")
	}
	tmpl, err := template.New(cfg.Name).Parse(cfg.PromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template: %w", err)
	}
	return &LLMProcessor{
		name:         cfg.Name,
		config:       *cfg,
		client:       client,
		defaultModel: defaultModel,
		template:     tmpl,
	}, nil
}

func (p *LLMProcessor) Name() string {
	return p.name
}

func (p *LLMProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *LLMProcessor) Validate() error {
	if p.client == nil {
		return fmt.Errorf("llm client is required")
	}
	if p.config.PromptTemplate == "" {
		return fmt.Errorf("prompt template is required")
	}
	return nil
}

func (p *LLMProcessor) Evaluate(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	filtered := make([]*core.PostBlock, 0, len(blocks))
	threshold := p.config.Threshold
	if threshold == 0 {
		threshold = 0.5
	}

	for _, block := range blocks {
		prompt, err := renderTemplate(p.template, block)
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

		var parsed qualityResponse
		if err := json.Unmarshal([]byte(response.Content), &parsed); err != nil {
			return nil, fmt.Errorf("parse llm quality response: %w", err)
		}

		result := "drop"
		if parsed.Score >= threshold {
			result = "pass"
		}
		block.Quality = &core.QualityResult{
			ProcessorName: p.name,
			Result:        result,
			Score:         parsed.Score,
			Reason:        parsed.Reason,
			ProcessedAt:   time.Now().UTC(),
		}
		if result == "pass" {
			filtered = append(filtered, block)
		}
	}

	return filtered, nil
}

func renderTemplate(tmpl *template.Template, data interface{}) (string, error) {
	builder := &strings.Builder{}
	if err := tmpl.Execute(builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}
