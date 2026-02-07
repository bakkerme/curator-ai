package source

import (
	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

func summaryPlanFromConfig(cfg *config.SummaryPlanConfig) *core.SummaryPlan {
	// Default summary behavior is full-document summarization when no plan is configured.
	if cfg == nil {
		return &core.SummaryPlan{Mode: core.SummaryModeFull}
	}

	mode := cfg.Mode
	if mode == "" {
		mode = core.SummaryModeFull
	}
	return &core.SummaryPlan{
		Mode:          mode,
		MaxChunkChars: cfg.MaxChunkChars,
		ChunkLimit:    cfg.ChunkLimit,
	}
}
