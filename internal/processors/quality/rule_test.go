package quality

import (
	"context"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

func TestRuleProcessorCompilesWithCapitalizedFields(t *testing.T) {
	cfg := &config.QualityRule{
		Name:       "comments_rule",
		Rule:       "Comments.count > 1",
		ActionType: "pass_drop",
		Result:     "drop",
	}

	processor, err := NewRuleProcessor(cfg)
	if err != nil {
		t.Fatalf("expected rule to compile, got error: %v", err)
	}

	blocks := []*core.PostBlock{
		{ID: "1", Comments: []core.CommentBlock{{ID: "c1"}}},
		{ID: "2", Comments: []core.CommentBlock{{ID: "c1"}, {ID: "c2"}}},
	}

	filtered, err := processor.Evaluate(context.Background(), blocks)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 block after filtering, got %d", len(filtered))
	}
	if filtered[0].ID != "1" {
		t.Errorf("expected block 1 to remain, got %s", filtered[0].ID)
	}
}

func TestRuleProcessorEvaluatesTitleLength(t *testing.T) {
	cfg := &config.QualityRule{
		Name:       "title_length",
		Rule:       "title.length > 5",
		ActionType: "pass_drop",
		Result:     "drop",
	}

	processor, err := NewRuleProcessor(cfg)
	if err != nil {
		t.Fatalf("expected rule to compile, got error: %v", err)
	}

	blocks := []*core.PostBlock{
		{ID: "short", Title: "abc"},
		{ID: "long", Title: "longer title"},
	}

	filtered, err := processor.Evaluate(context.Background(), blocks)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 block after filtering, got %d", len(filtered))
	}
	if filtered[0].ID != "short" {
		t.Errorf("expected short title to remain, got %s", filtered[0].ID)
	}
}
