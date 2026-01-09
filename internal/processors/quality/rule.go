package quality

import (
	"context"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

type RuleProcessor struct {
	name   string
	config config.QualityRule
	prg    cel.Program
}

func NewRuleProcessor(cfg *config.QualityRule) (core.QualityProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("quality rule config is required")
	}
	processor, err := newCELRuleProcessor(cfg)
	if err != nil {
		return nil, err
	}
	return processor, nil
}

func newCELRuleProcessor(cfg *config.QualityRule) (*RuleProcessor, error) {
	env, err := cel.NewEnv(
		cel.Variable("title", cel.StringType),
		cel.Variable("content", cel.StringType),
		cel.Variable("author", cel.StringType),
		cel.Variable("url", cel.StringType),
		cel.Variable("created_at", cel.TimestampType),
		cel.Variable("comment_count", cel.IntType),
		cel.Variable("title_length", cel.IntType),
		cel.Variable("content_length", cel.IntType),
	)
	if err != nil {
		return nil, fmt.Errorf("create CEL env: %w", err)
	}
	ast, iss := env.Compile(cfg.Rule)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("compile CEL rule: %w", iss.Err())
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("rule must return boolean, got %v", ast.OutputType())
	}
	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("create CEL program: %w", err)
	}
	return &RuleProcessor{
		name:   cfg.Name,
		config: *cfg,
		prg:    prg,
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
		activation := map[string]interface{}{
			"title":          block.Title,
			"content":        block.Content,
			"author":         block.Author,
			"url":            block.URL,
			"created_at":     block.CreatedAt,
			"comment_count":  int64(len(block.Comments)),
			"title_length":   int64(len(block.Title)),
			"content_length": int64(len(block.Content)),
		}

		out, _, err := p.prg.Eval(activation)
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
		matched, ok := out.Value().(bool)
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
