package quality

import (
	"context"
	"encoding/json"
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

type LLMProcessor struct {
	name           string
	config         config.LLMQuality
	client         llm.Client
	defaultModel   string
	defaultTemp    *float64
	systemTemplate *template.Template
	template       *template.Template
	logger         *slog.Logger
}

type qualityResponse struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

func NewLLMProcessor(cfg *config.LLMQuality, client llm.Client, defaultModel string) (*LLMProcessor, error) {
	return NewLLMProcessorWithLogger(cfg, client, defaultModel, nil, nil)
}

func NewLLMProcessorWithLogger(cfg *config.LLMQuality, client llm.Client, defaultModel string, logger *slog.Logger, defaultTemp *float64) (*LLMProcessor, error) {
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
		defaultTemp:    defaultTemp,
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

	decodeRetries := p.config.InvalidJSONRetries
	if decodeRetries == 0 {
		decodeRetries = RETRIES
	}

	evaluateOne := func(ctx context.Context, block *core.PostBlock) (bool, error) {
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
			return false, err
		}

		user_prompt, err := llmutil.ExecuteTemplate(p.template, data)
		if err != nil {
			return false, err
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
		logger.Info("llm quality evaluating block", "block_id", block.ID, "model", model, "temperature", temperatureLog)

		var parsed qualityResponse
		_, err = llmutil.ChatSystemUserWithRetries(
			ctx,
			p.client,
			model,
			system_prompt,
			user_prompt,
			decodeRetries,
			func(content string) error {
				return json.Unmarshal([]byte(content), &parsed)
			},
			temperature,
		)
		if err != nil {
			return false, fmt.Errorf("parse llm quality response: %w", err)
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
		return result == "pass", nil
	}

	maxConcurrency := p.config.MaxConcurrency
	if maxConcurrency <= 1 || len(blocks) <= 1 {
		for _, block := range blocks {
			pass, err := evaluateOne(ctx, block)
			if err != nil {
				return nil, err
			}
			if pass {
				filtered = append(filtered, block)
			}
		}
		return filtered, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	passResults := make([]bool, len(blocks))

loop:
	for i, block := range blocks {
		i, block := i, block
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			pass, err := evaluateOne(ctx, block)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
				return
			}
			passResults[i] = pass
		}()
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
		for i, block := range blocks {
			if passResults[i] {
				filtered = append(filtered, block)
			}
		}
		return filtered, nil
	}

	return filtered, nil
}
