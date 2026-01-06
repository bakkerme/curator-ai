package quality

import (
	"context"
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

type RuleProcessor struct {
	name    string
	config  config.QualityRule
	program *vm.Program
}

func NewRuleProcessor(cfg *config.QualityRule) (*RuleProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("quality rule config is required")
	}
	program, err := expr.Compile(cfg.Rule, expr.Env(sampleRuleEnv()))
	if err != nil {
		return nil, fmt.Errorf("compile quality rule: %w", err)
	}
	return &RuleProcessor{
		name:    cfg.Name,
		config:  *cfg,
		program: program,
	}, nil
}

func (p *RuleProcessor) Name() string {
	return p.name
}

func (p *RuleProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *RuleProcessor) Validate() error {
	if p.config.Name == "" || p.config.Rule == "" {
		return fmt.Errorf("rule name and expression are required")
	}
	return nil
}

func (p *RuleProcessor) Evaluate(ctx context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	_ = ctx
	if err := p.Validate(); err != nil {
		return nil, err
	}
	filtered := make([]*core.PostBlock, 0, len(blocks))

	for _, block := range blocks {
		env := qualityEnv(block)
		result, err := expr.Run(p.program, env)
		if err != nil {
			block.Errors = append(block.Errors, core.ProcessError{
				ProcessorName: p.name,
				Stage:         "quality",
				Error:         err.Error(),
				OccurredAt:    time.Now().UTC(),
			})
			filtered = append(filtered, block)
			continue
		}
		matched, ok := result.(bool)
		if !ok {
			return nil, fmt.Errorf("quality rule did not return bool")
		}

		shouldDrop := matched && p.config.Result == "drop"
		block.Quality = &core.QualityResult{
			ProcessorName: p.name,
			Result:        "pass",
			ProcessedAt:   time.Now().UTC(),
		}
		if shouldDrop {
			block.Quality.Result = "drop"
			continue
		}
		filtered = append(filtered, block)
	}

	return filtered, nil
}

func qualityEnv(block *core.PostBlock) map[string]interface{} {
	comments := map[string]interface{}{
		"count": len(block.Comments),
		"Count": len(block.Comments),
	}
	return map[string]interface{}{
		"title": map[string]interface{}{
			"value":  block.Title,
			"length": len(block.Title),
			"Value":  block.Title,
			"Length": len(block.Title),
		},
		"Title": map[string]interface{}{
			"value":  block.Title,
			"length": len(block.Title),
			"Value":  block.Title,
			"Length": len(block.Title),
		},
		"content": map[string]interface{}{
			"value":  block.Content,
			"length": len(block.Content),
			"Value":  block.Content,
			"Length": len(block.Content),
		},
		"Content": map[string]interface{}{
			"value":  block.Content,
			"length": len(block.Content),
			"Value":  block.Content,
			"Length": len(block.Content),
		},
		"author":     block.Author,
		"Author":     block.Author,
		"url":        block.URL,
		"URL":        block.URL,
		"comments":   comments,
		"Comments":   comments,
		"created_at": block.CreatedAt,
		"CreatedAt":  block.CreatedAt,
	}
}

func sampleRuleEnv() map[string]interface{} {
	comments := map[string]interface{}{
		"count": 0,
		"Count": 0,
	}
	return map[string]interface{}{
		"title": map[string]interface{}{
			"value":  "",
			"length": 0,
			"Value":  "",
			"Length": 0,
		},
		"Title": map[string]interface{}{
			"value":  "",
			"length": 0,
			"Value":  "",
			"Length": 0,
		},
		"content": map[string]interface{}{
			"value":  "",
			"length": 0,
			"Value":  "",
			"Length": 0,
		},
		"Content": map[string]interface{}{
			"value":  "",
			"length": 0,
			"Value":  "",
			"Length": 0,
		},
		"author":     "",
		"Author":     "",
		"url":        "",
		"URL":        "",
		"comments":   comments,
		"Comments":   comments,
		"created_at": time.Time{},
		"CreatedAt":  time.Time{},
	}
}
