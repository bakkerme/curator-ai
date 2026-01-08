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
	defaultTopP    *float64
	defaultPresPen *float64
	defaultTopK    *int
	systemTemplate *template.Template
	template       *template.Template
	logger         *slog.Logger
}

func NewRunLLMProcessor(cfg *config.LLMSummary, client llm.Client, defaultModel string) (*RunLLMProcessor, error) {
	return NewRunLLMProcessorWithLogger(cfg, client, defaultModel, nil, nil, nil, nil, nil)
}

func NewRunLLMProcessorWithLogger(
	cfg *config.LLMSummary,
	client llm.Client,
	defaultModel string,
	logger *slog.Logger,
	defaultTemp, defaultTopP, defaultPresencePenalty *float64,
	defaultTopK *int,
) (*RunLLMProcessor, error) {
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
		defaultTopP:    defaultTopP,
		defaultPresPen: defaultPresencePenalty,
		defaultTopK:    defaultTopK,
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
	topP := p.config.TopP
	if topP == nil {
		topP = p.defaultTopP
	}
	presencePenalty := p.config.PresencePenalty
	if presencePenalty == nil {
		presencePenalty = p.defaultPresPen
	}
	topK := p.config.TopK
	if topK == nil {
		topK = p.defaultTopK
	}

	attrs := []any{"blocks", len(blocks), "model", model, "has_current_summary", current != nil}
	if temperature != nil {
		attrs = append(attrs, "temperature", *temperature)
	}
	if topP != nil {
		attrs = append(attrs, "top_p", *topP)
	}
	if presencePenalty != nil {
		attrs = append(attrs, "presence_penalty", *presencePenalty)
	}
	if topK != nil {
		attrs = append(attrs, "top_k", *topK)
	}
	logger.Info("llm run summary summarizing", attrs...)

	var summary string
	_, err = llmutil.ChatCompletionWithRetries(
		ctx,
		p.client,
		llm.ChatRequest{
			Model: model,
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: system_prompt},
				{Role: llm.RoleUser, Content: user_prompt},
			},
			Temperature:     temperature,
			TopP:            topP,
			PresencePenalty: presencePenalty,
			TopK:            topK,
		},
		RUN_RETRIES,
		func(content string) error {
			summary = content
			return nil
		},
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
