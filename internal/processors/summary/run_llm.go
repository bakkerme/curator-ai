package summary

import (
	"context"
	"fmt"
	"log/slog"
	"text/template"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
	"github.com/bakkerme/curator-ai/internal/processors/llmutil"
)

var RUN_RETRIES = 3

type RunLLMProcessor struct {
	name           string
	config         config.LLMSummary
	client         llm.Client
	defaultModel   string
	defaultTemp    *float64
	systemTemplate *template.Template
	template       *template.Template
	logger         *slog.Logger
}

func NewRunLLMProcessor(cfg *config.LLMSummary, client llm.Client, defaultModel string) (*RunLLMProcessor, error) {
	return NewRunLLMProcessorWithLogger(cfg, client, defaultModel, nil, nil)
}

func NewRunLLMProcessorWithLogger(cfg *config.LLMSummary, client llm.Client, defaultModel string, logger *slog.Logger, defaultTemp *float64) (*RunLLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("run summary config is required")
	}

	systemTmpl, tmpl, err := llmutil.ParseSystemAndPromptTemplates(cfg.Name, cfg.SystemTemplate, cfg.PromptTemplate)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &RunLLMProcessor{
		name:           cfg.Name,
		config:         *cfg,
		client:         client,
		defaultModel:   defaultModel,
		defaultTemp:    defaultTemp,
		systemTemplate: systemTmpl,
		template:       tmpl,
		logger:         logger,
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
		return fmt.Errorf("run summary context must be flow.")
	}
	if p.config.PromptTemplate == "" {
		return fmt.Errorf("prompt template is required for llm run summary processor")
	}
	if p.config.SystemTemplate == "" {
		return fmt.Errorf("system template is required for llm run summary processor")
	}

	return nil
}

func (p *RunLLMProcessor) SummarizeRun(ctx context.Context, blocks []*core.PostBlock, current *core.RunSummary) (*core.RunSummary, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	logger := p.logger
	if ctxLogger := core.LoggerFromContext(ctx); ctxLogger != nil {
		logger = ctxLogger
	}
	logger = logger.With("processor", p.name, "processor_type", fmt.Sprintf("%T", p))

	data := struct {
		Blocks []*core.PostBlock
		Params map[string]interface{}
	}{
		Blocks: blocks,
		Params: p.config.Params,
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
	temperature := p.config.Temperature
	if temperature == nil {
		temperature = p.defaultTemp
	}
	var temperatureLog any
	if temperature != nil {
		temperatureLog = *temperature
	}
	logger.Info("llm run summary summarizing", "blocks", len(blocks), "model", model, "temperature", temperatureLog, "has_current_summary", current != nil)

	var summary string
	_, err = llmutil.ChatSystemUserWithRetries(
		ctx,
		p.client,
		model,
		system_prompt,
		user_prompt,
		RUN_RETRIES,
		func(content string) error {
			summary = content
			return nil
		},
		temperature,
	)
	if err != nil {
		return nil, err
	}

	return &core.RunSummary{
		ProcessorName: p.name,
		Summary:       summary,
		PostCount:     len(blocks),
		ProcessedAt:   time.Now().UTC(),
	}, nil
}
