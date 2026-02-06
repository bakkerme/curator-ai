package source

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

func TestTestFileProcessorFetchesChunks(t *testing.T) {
	path := filepath.Join("testdata", "sample.md")
	cfg := &config.TestFileSource{
		Path:        path,
		ChunkSize:   10,
		SummaryPlan: &config.SummaryPlanConfig{Mode: core.SummaryModePerChunk},
	}
	processor, err := NewTestFileProcessor(cfg)
	if err != nil {
		t.Fatalf("NewTestFileProcessor error: %v", err)
	}
	blocks, err := processor.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].SummaryPlan == nil {
		t.Fatalf("expected summary plan to be set")
	}
	if len(blocks[0].Chunks) == 0 {
		t.Fatalf("expected chunks to be populated")
	}
}
