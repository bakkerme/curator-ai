package runner

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/runner/snapshot"
)

type testSource struct {
	name string
}

func (s *testSource) Name() string { return s.name }
func (s *testSource) Configure(map[string]interface{}) error { return nil }
func (s *testSource) Validate() error { return nil }
func (s *testSource) Fetch(context.Context) ([]*core.PostBlock, error) {
	return []*core.PostBlock{{ID: "p1"}}, nil
}

type testQuality struct {
	name       string
	evaluateFn func([]*core.PostBlock) ([]*core.PostBlock, error)
}

func (q *testQuality) Name() string { return q.name }
func (q *testQuality) Configure(map[string]interface{}) error { return nil }
func (q *testQuality) Validate() error { return nil }
func (q *testQuality) Evaluate(_ context.Context, blocks []*core.PostBlock) ([]*core.PostBlock, error) {
	return q.evaluateFn(blocks)
}

func TestRunner_QualityRestore_HonorsLaterRestoreWithoutRunningProcessor(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	rulePath := filepath.Join(tmp, "quality-rule.json")
	llmPath := filepath.Join(tmp, "quality-llm.json")

	ruleBlocks := []*core.PostBlock{{ID: "rule-1"}}
	llmBlocks := []*core.PostBlock{{ID: "llm-1"}, {ID: "llm-2"}}

	if err := snapshot.Save(rulePath, ruleBlocks, nil); err != nil {
		t.Fatalf("save rule snapshot: %v", err)
	}
	if err := snapshot.Save(llmPath, llmBlocks, nil); err != nil {
		t.Fatalf("save llm snapshot: %v", err)
	}

	llmEvaluated := 0
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := New(logger)
	flow := &core.Flow{
		ID:      "flow-1",
		Sources: []core.SourceProcessor{&testSource{name: "source"}},
		Quality: []core.QualityProcessor{
			snapshot.WrapQuality(&testQuality{name: "quality_rule", evaluateFn: func(blocks []*core.PostBlock) ([]*core.PostBlock, error) {
				t.Fatalf("quality_rule Evaluate should not run when restore is enabled")
				return nil, nil
			}}, &core.SnapshotConfig{Restore: true, Path: rulePath}),
			snapshot.WrapQuality(&testQuality{name: "quality_llm", evaluateFn: func(blocks []*core.PostBlock) ([]*core.PostBlock, error) {
				llmEvaluated++
				return blocks, nil
			}}, &core.SnapshotConfig{Restore: true, Path: llmPath}),
		},
	}

	run, err := r.RunOnce(context.Background(), flow)
	if err != nil {
		t.Fatalf("RunOnce error: %v", err)
	}
	if run.Status != core.RunStatusCompleted {
		t.Fatalf("unexpected run status: %s", run.Status)
	}
	if llmEvaluated != 0 {
		t.Fatalf("quality_llm should not have been evaluated, got %d calls", llmEvaluated)
	}
	if got, want := len(run.Blocks), len(llmBlocks); got != want {
		t.Fatalf("unexpected blocks length: got %d want %d", got, want)
	}
	if got, want := run.Blocks[0].ID, llmBlocks[0].ID; got != want {
		t.Fatalf("unexpected first block: got %q want %q", got, want)
	}
}

