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
	imageSystem    *template.Template
	imageTemplate  *template.Template
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
	var imageSystemTmpl *template.Template
	var imagePromptTmpl *template.Template
	if cfg.Images != nil && cfg.Images.Enabled && cfg.Images.Mode == config.ImageModeCaption {
		if cfg.Images.Caption == nil {
			return nil, fmt.Errorf("image caption config is required when images.mode=caption")
		}
		imageSystemTmpl, imagePromptTmpl, err = llmutil.ParseSystemAndPromptTemplates(cfg.Name+"-image-caption", cfg.Images.Caption.SystemTemplate, cfg.Images.Caption.PromptTemplate)
		if err != nil {
			return nil, err
		}
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
		imageSystem:    imageSystemTmpl,
		imageTemplate:  imagePromptTmpl,
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
	policy := p.config.BlockErrorPolicy
	if policy == "" {
		policy = config.BlockErrorPolicyFail
	}

	summarizeOne := func(ctx context.Context, block *core.PostBlock) error {
		if err := llmutil.EnsureImageCaptions(
			ctx,
			p.client,
			block,
			p.config.Images,
			llmutil.CaptionTemplates{System: p.imageSystem, Prompt: p.imageTemplate},
			p.defaultModel,
			p.defaultTemp,
			logger,
		); err != nil {
			return err
		}

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

		var response llm.ChatResponse
		if p.config.Images != nil && p.config.Images.Enabled && p.config.Images.Mode == config.ImageModeMultimodal {
			images := llmutil.CollectImageBlocks(block, p.config.Images.IncludeCommentImages, p.config.Images.MaxImages)
			// Multimodal calls are brittle in the face of dead/expired image URLs (common for scraped content).
			// Some providers fail the *entire* request if any referenced image returns 404.
			//
			// We treat this as a recoverable error: if we can detect this specific failure, we retry the same
			// prompt while progressively removing the offending image(s) so the post summary still completes.
			for {
				userMessage := llmutil.BuildUserMessageWithImages(userPrompt, images)
				response, err = llmutil.ChatCompletionWithRetries(ctx, p.client, model, []llm.Message{
					{Role: llm.RoleSystem, Content: systemPrompt},
					userMessage,
				}, RETRIES, nil, temperature)
				if err == nil {
					break
				}
				if url, ok := llmutil.MissingImageURL(err); ok && len(images) > 0 {
					// MissingImageURL is a best-effort heuristic based on parsing the upstream provider error
					// message (see llmutil.MissingImageURL docs). We only retry if we can identify and remove
					// the exact failing image URL; otherwise we fail the block and let block_error_policy decide
					// whether the post should be dropped.
					if url == "" {
						break
					}
					var removed *core.ImageBlock
					images, removed = llmutil.DropImageByURL(images, url)
					if removed == nil {
						break
					}
					logger.Warn("llm post summary missing image; retrying without image", "block_id", block.ID, "image_url", removed.URL)
					continue
				}
				break
			}
		} else {
			response, err = llmutil.ChatSystemUserWithRetries(ctx, p.client, model, systemPrompt, userPrompt, RETRIES, nil, temperature)
		}
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
		if policy == config.BlockErrorPolicyDrop {
			filtered := make([]*core.PostBlock, 0, len(blocks))
			for _, block := range blocks {
				if err := summarizeOne(ctx, block); err != nil {
					logger.Warn("llm post summary failed for block (dropping)", "block_id", block.ID, "error", err)
					continue
				}
				filtered = append(filtered, block)
			}
			return filtered, nil
		}

		for _, block := range blocks {
			if err := summarizeOne(ctx, block); err != nil {
				return nil, fmt.Errorf("llm post summary failed for block. Failing due to block_error_policy being set to fail. To ignore, change policy to drop. Block ID: %s, error: %w", block.ID, err)
			}
		}
		return blocks, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	keep := make([]bool, len(blocks))

loop:
	for i, block := range blocks {
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
				if policy == config.BlockErrorPolicyDrop {
					logger.Warn("llm post summary failed for block (dropping)", "block_id", block.ID, "error", err)
					keep[i] = false
					return
				}
				select {
				case errCh <- fmt.Errorf("llm post summary failed for block. Failing due to block_error_policy being set to fail. To ignore, change policy to drop. Block ID: %s, error: %w", block.ID, err):
				default:
				}
				cancel()
				return
			}
			keep[i] = true
		}()
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
		if policy == config.BlockErrorPolicyDrop {
			filtered := make([]*core.PostBlock, 0, len(blocks))
			for i, block := range blocks {
				if keep[i] {
					filtered = append(filtered, block)
				}
			}
			return filtered, nil
		}
		return blocks, nil
	}
}
