package source

import "github.com/bakkerme/curator-ai/internal/config"
import "github.com/bakkerme/curator-ai/internal/core"

func summaryPlanFromConfig(cfg *config.SummaryPlanConfig) *core.SummaryPlan {
	if cfg == nil {
		return nil
	}
	return &core.SummaryPlan{
		Mode:          cfg.Mode,
		MaxChunkChars: cfg.MaxChunkChars,
		ChunkLimit:    cfg.ChunkLimit,
	}
}
