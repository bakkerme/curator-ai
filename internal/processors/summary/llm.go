package summary

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	defaultTemp    *float64
	systemTemplate *template.Template
	template       *template.Template
	logger         *slog.Logger
}

func NewPostLLMProcessor(cfg *config.LLMSummary, client llm.Client, defaultModel string) (*PostLLMProcessor, error) {
	return NewPostLLMProcessorWithLogger(cfg, client, defaultModel, nil, nil)
}

func NewPostLLMProcessorWithLogger(cfg *config.LLMSummary, client llm.Client, defaultModel string, logger *slog.Logger, defaultTemp *float64) (*PostLLMProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("summary config is required")
	}

	systemTmpl, tmpl, err := llmutil.ParseSystemAndPromptTemplates(cfg.Name, cfg.SystemTemplate, cfg.PromptTemplate)
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &PostLLMProcessor{
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
	logger := p.logger
	if ctxLogger := core.LoggerFromContext(ctx); ctxLogger != nil {
		logger = ctxLogger
	}
	logger = logger.With("processor", p.name, "processor_type", fmt.Sprintf("%T", p))

	summarizeOne := func(ctx context.Context, block *core.PostBlock) error {
		data := struct {
			*core.PostBlock
			Params map[string]interface{}
		}{
			PostBlock: block,
			Params:    p.config.Params,
		}

		systemPrompt, err := llmutil.ExecuteTemplate(p.systemTemplate, data)
		if err != nil {
			return err
		}

		userPrompt, err := llmutil.ExecuteTemplate(p.template, data)
		if err != nil {
			return err
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
		logger.Info("llm post summary summarizing block", "block_id", block.ID, "model", model, "temperature", temperatureLog)

		response, err := llmutil.ChatSystemUserWithRetries(ctx, p.client, model, systemPrompt, userPrompt, RETRIES, nil, temperature)
		if err != nil {
			return err
		}
		block.Summary = &core.SummaryResult{
			ProcessorName: p.name,
			Summary:       response.Content,
			ProcessedAt:   time.Now().UTC(),
		}
		return nil
	}

	maxConcurrency := p.config.MaxConcurrency
	if maxConcurrency <= 1 || len(blocks) <= 1 {
		for _, block := range blocks {
			if err := summarizeOne(ctx, block); err != nil {
				return nil, err
			}
		}
		return blocks, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

loop:
	for _, block := range blocks {
		block := block
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := summarizeOne(ctx, block); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
		return blocks, nil
	}
}
