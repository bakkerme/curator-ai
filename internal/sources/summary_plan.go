package sources

import (
	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

// SummaryPlanFromConfig normalizes a source summary plan config into the runtime
// form attached to PostBlocks.
func SummaryPlanFromConfig(cfg *config.SummaryPlanConfig) *core.SummaryPlan {
	// Sources default to whole-document summarization when no explicit plan is
	// configured.
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
