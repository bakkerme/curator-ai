package sources

import (
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

func TestSummaryPlanFromConfig_DefaultsWhenConfigIsNil(t *testing.T) {
	plan := SummaryPlanFromConfig(nil)

	if plan == nil {
		t.Fatal("expected summary plan, got nil")
	}
	if plan.Mode != core.SummaryModeFull {
		t.Fatalf("expected mode %q, got %q", core.SummaryModeFull, plan.Mode)
	}
	if plan.MaxChunkChars != 0 {
		t.Fatalf("expected MaxChunkChars 0, got %d", plan.MaxChunkChars)
	}
	if plan.ChunkLimit != 0 {
		t.Fatalf("expected ChunkLimit 0, got %d", plan.ChunkLimit)
	}
}

func TestSummaryPlanFromConfig_DefaultsMissingMode(t *testing.T) {
	plan := SummaryPlanFromConfig(&config.SummaryPlanConfig{
		MaxChunkChars: 1200,
		ChunkLimit:    4,
	})

	if plan.Mode != core.SummaryModeFull {
		t.Fatalf("expected mode %q, got %q", core.SummaryModeFull, plan.Mode)
	}
	if plan.MaxChunkChars != 1200 {
		t.Fatalf("expected MaxChunkChars 1200, got %d", plan.MaxChunkChars)
	}
	if plan.ChunkLimit != 4 {
		t.Fatalf("expected ChunkLimit 4, got %d", plan.ChunkLimit)
	}
}

func TestSummaryPlanFromConfig_PreservesConfiguredValues(t *testing.T) {
	plan := SummaryPlanFromConfig(&config.SummaryPlanConfig{
		Mode:          core.SummaryModeMapReduce,
		MaxChunkChars: 2400,
		ChunkLimit:    8,
	})

	if plan.Mode != core.SummaryModeMapReduce {
		t.Fatalf("expected mode %q, got %q", core.SummaryModeMapReduce, plan.Mode)
	}
	if plan.MaxChunkChars != 2400 {
		t.Fatalf("expected MaxChunkChars 2400, got %d", plan.MaxChunkChars)
	}
	if plan.ChunkLimit != 8 {
		t.Fatalf("expected ChunkLimit 8, got %d", plan.ChunkLimit)
	}
}
