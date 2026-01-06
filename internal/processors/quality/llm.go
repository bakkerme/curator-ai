package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
	"github.com/bakkerme/curator-ai/internal/processors/llmutil"
)

var RETRIES = 3

type LLMProcessor struct {
	name           string
	config         config.LLMQuality
	client         llm.Client
	defaultModel   string
	systemTemplate *template.Template
	template       *template.Template
	logger         *slog.Logger
}

type qualityResponse struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

func NewLLMProcessor(cfg *config.LLMQuality, client llm.Client, defaultModel string) (*LLMProcessor, error) {
	return NewLLMProcessorWithLogger(cfg, client, defaultModel, nil)
}

func NewLLMProcessorWithLogger(cfg *config.LLMQuality, client llm.Client, defaultModel string, logger *slog.Logger) (*LLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm quality config is required")
	}

	systemTmpl, tmpl, err := llmutil.ParseSystemAndPromptTemplates(cfg.Name, cfg.SystemTemplate, cfg.PromptTemplate)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &LLMProcessor{
		name:           cfg.Name,
		config:         *cfg,
		client:         client,
		defaultModel:   defaultModel,
		systemTemplate: systemTmpl,
		template:       tmpl,
		logger:         logger,
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
		return fmt.Errorf("llm client is required for llm quality processor")
	}
	if p.config.PromptTemplate == "" {
		return fmt.Errorf("prompt template is required for llm quality processor")
	}
	if p.config.SystemTemplate == "" {
		return fmt.Errorf("system template is required for llm quality processor")
	}
	return nil
}

func (p *LLMProcessor) Evaluate(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	logger := p.logger
	if ctxLogger := core.LoggerFromContext(ctx); ctxLogger != nil {
		logger = ctxLogger
	}
	logger = logger.With("processor", p.name, "processor_type", fmt.Sprintf("%T", p))

	filtered := make([]*core.PostBlock, 0, len(blocks))
	threshold := p.config.Threshold
	if threshold == 0 {
		threshold = 0.5
	}

	for _, block := range blocks {
		data := struct {
			*core.PostBlock
			Evaluations []string
			Exclusions  []string
		}{
			PostBlock:   block,
			Evaluations: p.config.Evaluations,
			Exclusions:  p.config.Exclusions,
		}

		system_prompt, err := llmutil.ExecuteTemplate(p.systemTemplate, data)
		if err != nil {
			return nil, err
		}

		user_prompt, err := llmutil.ExecuteTemplate(p.template, data)
		if err != nil {
			return nil, err
		}

		model := llmutil.ModelOrDefault(p.config.Model, p.defaultModel)
		logger.Info("llm quality evaluating block", "block_id", block.ID, "model", model)

		var parsed qualityResponse
		_, err = llmutil.ChatSystemUserWithRetries(
			ctx,
			p.client,
			model,
			system_prompt,
			user_prompt,
			RETRIES,
			func(content string) error {
				return json.Unmarshal([]byte(content), &parsed)
			},
		)
		if err != nil {
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
