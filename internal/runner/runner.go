package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/runner/snapshot"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func logStage(logger *slog.Logger, stage string, processorName string, processorType string, before int, after int, duration time.Duration) {
	delta := after - before
	removed := 0
	if before > after {
		removed = before - after
	}

	logger.Info(
		"stage completed",
		"stage", stage,
		"processor", processorName,
		"processor_type", processorType,
		"blocks_before", before,
		"blocks_after", after,
		"blocks_delta", delta,
		"blocks_removed", removed,
		"duration", duration,
	)
}

type Runner struct {
	logger                   *slog.Logger
	allowPartialSourceErrors bool
}

func snapshotConfig(processor interface{}) *core.SnapshotConfig {
	if provider, ok := processor.(snapshot.ConfigProvider); ok {
		return provider.SnapshotConfig()
	}
	return nil
}

func findSourceRestoreIndex(processors []core.SourceProcessor) (int, *core.SnapshotConfig) {
	for i, processor := range processors {
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			return i, cfg
		}
	}
	return -1, nil
}

func findQualityRestoreIndex(processors []core.QualityProcessor) (int, *core.SnapshotConfig) {
	for i, processor := range processors {
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			return i, cfg
		}
	}
	return -1, nil
}

func findSummaryRestoreIndex(processors []core.SummaryProcessor) (int, *core.SnapshotConfig) {
	for i, processor := range processors {
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			return i, cfg
		}
	}
	return -1, nil
}

func findRunSummaryRestoreIndex(processors []core.RunSummaryProcessor) (int, *core.SnapshotConfig) {
	for i, processor := range processors {
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			return i, cfg
		}
	}
	return -1, nil
}

func findOutputRestoreIndex(processors []core.OutputProcessor) (int, *core.SnapshotConfig) {
	for i, processor := range processors {
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			return i, cfg
		}
	}
	return -1, nil
}

func processorName(processor interface{}, fallback string) string {
	if processor == nil {
		return fallback
	}
	if named, ok := processor.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fallback
}

type Config struct {
	AllowPartialSourceErrors bool
}

func New(logger *slog.Logger) *Runner {
	return NewWithConfig(logger, Config{})
}

func NewWithConfig(logger *slog.Logger, config Config) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		logger:                   logger,
		allowPartialSourceErrors: config.AllowPartialSourceErrors,
	}
}

func (r *Runner) Start(ctx context.Context, flow *core.Flow) error {
	if flow == nil {
		return fmt.Errorf("flow is required")
	}
	for _, trigger := range flow.Triggers {
		if trigger == nil {
			continue
		}
		events, err := trigger.Start(ctx, flow.ID)
		if err != nil {
			return err
		}
		go r.listen(ctx, flow, events)
	}
	return nil
}

func (r *Runner) RunOnce(ctx context.Context, flow *core.Flow) (*core.Run, error) {
	if flow == nil {
		return nil, fmt.Errorf("flow is required")
	}
	run := &core.Run{
		ID:        fmt.Sprintf("run-%d", time.Now().UnixNano()),
		FlowID:    flow.ID,
		StartedAt: time.Now().UTC(),
		Status:    core.RunStatusRunning,
	}

	logger := r.logger.With("flow_id", flow.ID, "run_id", run.ID)
	ctx = core.WithLogger(ctx, logger)
	ctx = core.WithFlowID(ctx, flow.ID)
	ctx = core.WithRunID(ctx, run.ID)

	tracer := otel.Tracer("curator-ai/runner")
	ctx, span := tracer.Start(
		ctx,
		"curator.run",
	)
	span.SetAttributes(
		attribute.String("flow.id", flow.ID),
		attribute.String("run.id", run.ID),
		attribute.String("session.id", run.ID),
	)
	defer span.End()

	logger.Info("run started")

	blocks := []*core.PostBlock{}
	var runSummary *core.RunSummary

	sourceStart := 0
	if restoreIndex, cfg := findSourceRestoreIndex(flow.Sources); cfg != nil {
		restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error("snapshot restore failed", "stage", "source", "processor", processorName(flow.Sources[restoreIndex], "source"), "path", cfg.Path, "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		blocks = restoredBlocks
		if restoredSummary != nil {
			runSummary = restoredSummary
		}
		sourceStart = restoreIndex + 1
		logger.Info("snapshot restored", "stage", "source", "processor", processorName(flow.Sources[restoreIndex], "source"), "path", cfg.Path, "blocks", len(blocks))
	}
	for i := sourceStart; i < len(flow.Sources); i++ {
		source := flow.Sources[i]
		if source == nil {
			continue
		}
		if cfg := snapshotConfig(source); cfg != nil && cfg.Restore {
			restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
			if err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot restore failed", "stage", "source", "processor", source.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			blocks = restoredBlocks
			if restoredSummary != nil {
				runSummary = restoredSummary
			}
			logger.Info("snapshot restored", "stage", "source", "processor", source.Name(), "path", cfg.Path, "blocks", len(blocks))
			continue
		}
		sourceName := source.Name()
		start := time.Now()
		logger.Info("stage started", "stage", "source", "processor", sourceName, "processor_type", fmt.Sprintf("%T", source), "blocks_before", len(blocks))
		fetched, err := source.Fetch(ctx)
		if err != nil {
			if r.allowPartialSourceErrors {
				logger.Warn(
					"source fetch failed (continuing due to allow_partial_source_errors)",
					"stage", "source",
					"processor", sourceName,
					"processor_type", fmt.Sprintf("%T", source),
					"error", err,
				)
				continue
			}
			run.Status = core.RunStatusFailed
			logger.Error(
				"source fetch failed",
				"stage", "source",
				"processor", sourceName,
				"processor_type", fmt.Sprintf("%T", source),
				"error", err,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		before := len(blocks)
		blocks = append(blocks, fetched...)
		logStage(logger, "source", sourceName, fmt.Sprintf("%T", source), before, len(blocks), time.Since(start))
		if cfg := snapshotConfig(source); cfg != nil && cfg.Snapshot {
			if err := snapshot.Save(cfg.Path, blocks, runSummary); err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot save failed", "stage", "source", "processor", sourceName, "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			logger.Info("snapshot saved", "stage", "source", "processor", sourceName, "path", cfg.Path, "blocks", len(blocks))
		}
	}
	logger.Info("sources completed", "blocks", len(blocks))

	if len(blocks) == 0 {
		logger.Info("source returned no blocks, skipping processing and outputs")
		run.Status = core.RunStatusCompleted
		return run, nil
	}

	qualityStart := 0
	if restoreIndex, cfg := findQualityRestoreIndex(flow.Quality); cfg != nil {
		restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error("snapshot restore failed", "stage", "quality", "processor", processorName(flow.Quality[restoreIndex], "quality"), "path", cfg.Path, "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		blocks = restoredBlocks
		if restoredSummary != nil {
			runSummary = restoredSummary
		}
		qualityStart = restoreIndex + 1
		logger.Info("snapshot restored", "stage", "quality", "processor", processorName(flow.Quality[restoreIndex], "quality"), "path", cfg.Path, "blocks", len(blocks))
	}
	for i := qualityStart; i < len(flow.Quality); i++ {
		processor := flow.Quality[i]
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
			if err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot restore failed", "stage", "quality", "processor", processor.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			blocks = restoredBlocks
			if restoredSummary != nil {
				runSummary = restoredSummary
			}
			logger.Info("snapshot restored", "stage", "quality", "processor", processor.Name(), "path", cfg.Path, "blocks", len(blocks))
			continue
		}
		before := len(blocks)
		start := time.Now()
		logger.Info("stage started", "stage", "quality", "processor", processor.Name(), "processor_type", fmt.Sprintf("%T", processor), "blocks_before", before)
		next, err := processor.Evaluate(ctx, blocks)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error(
				"quality processing failed",
				"stage", "quality",
				"processor", processor.Name(),
				"processor_type", fmt.Sprintf("%T", processor),
				"blocks_before", before,
				"error", err,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		blocks = next
		logStage(logger, "quality", processor.Name(), fmt.Sprintf("%T", processor), before, len(blocks), time.Since(start))
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Snapshot {
			if err := snapshot.Save(cfg.Path, blocks, runSummary); err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot save failed", "stage", "quality", "processor", processor.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			logger.Info("snapshot saved", "stage", "quality", "processor", processor.Name(), "path", cfg.Path, "blocks", len(blocks))
		}
	}

	if len(blocks) == 0 {
		logger.Info("no blocks left after quality processing, skipping summary and outputs")
		run.Status = core.RunStatusCompleted
		return run, nil
	}

	postSummaryStart := 0
	if restoreIndex, cfg := findSummaryRestoreIndex(flow.PostSummary); cfg != nil {
		restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error("snapshot restore failed", "stage", "post_summary", "processor", processorName(flow.PostSummary[restoreIndex], "post_summary"), "path", cfg.Path, "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		blocks = restoredBlocks
		if restoredSummary != nil {
			runSummary = restoredSummary
		}
		postSummaryStart = restoreIndex + 1
		logger.Info("snapshot restored", "stage", "post_summary", "processor", processorName(flow.PostSummary[restoreIndex], "post_summary"), "path", cfg.Path, "blocks", len(blocks))
	}
	for i := postSummaryStart; i < len(flow.PostSummary); i++ {
		processor := flow.PostSummary[i]
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
			if err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot restore failed", "stage", "post_summary", "processor", processor.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			blocks = restoredBlocks
			if restoredSummary != nil {
				runSummary = restoredSummary
			}
			logger.Info("snapshot restored", "stage", "post_summary", "processor", processor.Name(), "path", cfg.Path, "blocks", len(blocks))
			continue
		}
		before := len(blocks)
		start := time.Now()
		logger.Info("stage started", "stage", "post_summary", "processor", processor.Name(), "processor_type", fmt.Sprintf("%T", processor), "blocks_before", before)
		next, err := processor.Summarize(ctx, blocks)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error(
				"post summary processing failed",
				"stage", "post_summary",
				"processor", processor.Name(),
				"processor_type", fmt.Sprintf("%T", processor),
				"blocks_before", before,
				"error", err,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		blocks = next
		logStage(logger, "post_summary", processor.Name(), fmt.Sprintf("%T", processor), before, len(blocks), time.Since(start))
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Snapshot {
			if err := snapshot.Save(cfg.Path, blocks, runSummary); err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot save failed", "stage", "post_summary", "processor", processor.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			logger.Info("snapshot saved", "stage", "post_summary", "processor", processor.Name(), "path", cfg.Path, "blocks", len(blocks))
		}
	}

	runSummaryStart := 0
	if restoreIndex, cfg := findRunSummaryRestoreIndex(flow.RunSummary); cfg != nil {
		restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error("snapshot restore failed", "stage", "run_summary", "processor", processorName(flow.RunSummary[restoreIndex], "run_summary"), "path", cfg.Path, "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		if restoredBlocks != nil {
			blocks = restoredBlocks
		}
		runSummary = restoredSummary
		runSummaryStart = restoreIndex + 1
		logger.Info("snapshot restored", "stage", "run_summary", "processor", processorName(flow.RunSummary[restoreIndex], "run_summary"), "path", cfg.Path, "blocks", len(blocks), "has_summary", runSummary != nil)
	}
	for i := runSummaryStart; i < len(flow.RunSummary); i++ {
		processor := flow.RunSummary[i]
		if processor == nil {
			continue
		}
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Restore {
			restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
			if err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot restore failed", "stage", "run_summary", "processor", processor.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			if restoredBlocks != nil {
				blocks = restoredBlocks
			}
			runSummary = restoredSummary
			logger.Info("snapshot restored", "stage", "run_summary", "processor", processor.Name(), "path", cfg.Path, "blocks", len(blocks), "has_summary", runSummary != nil)
			continue
		}
		start := time.Now()
		logger.Info("stage started", "stage", "run_summary", "processor", processor.Name(), "processor_type", fmt.Sprintf("%T", processor), "blocks", len(blocks), "has_current_summary", runSummary != nil)
		summary, err := processor.SummarizeRun(ctx, blocks, runSummary)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error(
				"run summary processing failed",
				"stage", "run_summary",
				"processor", processor.Name(),
				"processor_type", fmt.Sprintf("%T", processor),
				"blocks", len(blocks),
				"error", err,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		runSummary = summary
		logger.Info(
			"run summary completed",
			"stage", "run_summary",
			"processor", processor.Name(),
			"processor_type", fmt.Sprintf("%T", processor),
			"blocks", len(blocks),
			"has_summary", runSummary != nil,
			"duration", time.Since(start),
		)
		if cfg := snapshotConfig(processor); cfg != nil && cfg.Snapshot {
			if err := snapshot.Save(cfg.Path, blocks, runSummary); err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot save failed", "stage", "run_summary", "processor", processor.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			logger.Info("snapshot saved", "stage", "run_summary", "processor", processor.Name(), "path", cfg.Path, "blocks", len(blocks), "has_summary", runSummary != nil)
		}
	}

	if len(blocks) == 0 {
		logger.Info("no blocks to deliver, skipping outputs")
		run.Status = core.RunStatusCompleted
		return run, nil
	}

	outputStart := 0
	if restoreIndex, cfg := findOutputRestoreIndex(flow.Outputs); cfg != nil {
		restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
		if err != nil {
			run.Status = core.RunStatusFailed
			logger.Error("snapshot restore failed", "stage", "output", "processor", processorName(flow.Outputs[restoreIndex], "output"), "path", cfg.Path, "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		if restoredBlocks != nil {
			blocks = restoredBlocks
		}
		if restoredSummary != nil {
			runSummary = restoredSummary
		}
		outputStart = restoreIndex + 1
		logger.Info("snapshot restored", "stage", "output", "processor", processorName(flow.Outputs[restoreIndex], "output"), "path", cfg.Path, "blocks", len(blocks), "has_run_summary", runSummary != nil)
	}
	for i := outputStart; i < len(flow.Outputs); i++ {
		output := flow.Outputs[i]
		if output == nil {
			continue
		}
		if cfg := snapshotConfig(output); cfg != nil && cfg.Restore {
			restoredBlocks, restoredSummary, err := snapshot.Load(cfg.Path)
			if err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot restore failed", "stage", "output", "processor", output.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			if restoredBlocks != nil {
				blocks = restoredBlocks
			}
			if restoredSummary != nil {
				runSummary = restoredSummary
			}
			logger.Info("snapshot restored", "stage", "output", "processor", output.Name(), "path", cfg.Path, "blocks", len(blocks), "has_run_summary", runSummary != nil)
			continue
		}
		start := time.Now()
		logger.Info("stage started", "stage", "output", "processor", output.Name(), "processor_type", fmt.Sprintf("%T", output), "blocks", len(blocks), "has_run_summary", runSummary != nil)
		if err := output.Deliver(ctx, blocks, runSummary); err != nil {
			run.Status = core.RunStatusFailed
			logger.Error(
				"output delivery failed",
				"stage", "output",
				"processor", output.Name(),
				"processor_type", fmt.Sprintf("%T", output),
				"blocks", len(blocks),
				"error", err,
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return run, err
		}
		logger.Info(
			"output delivery completed",
			"stage", "output",
			"processor", output.Name(),
			"processor_type", fmt.Sprintf("%T", output),
			"blocks", len(blocks),
			"duration", time.Since(start),
		)
		if cfg := snapshotConfig(output); cfg != nil && cfg.Snapshot {
			if err := snapshot.Save(cfg.Path, blocks, runSummary); err != nil {
				run.Status = core.RunStatusFailed
				logger.Error("snapshot save failed", "stage", "output", "processor", output.Name(), "path", cfg.Path, "error", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return run, err
			}
			logger.Info("snapshot saved", "stage", "output", "processor", output.Name(), "path", cfg.Path, "blocks", len(blocks), "has_run_summary", runSummary != nil)
		}
	}

	completedAt := time.Now().UTC()
	run.CompletedAt = &completedAt
	run.Status = core.RunStatusCompleted
	run.Blocks = blocks
	run.RunSummary = runSummary
	span.SetStatus(codes.Ok, "")
	logger.Info("run completed", "status", run.Status, "blocks", len(blocks), "has_run_summary", runSummary != nil)
	return run, nil
}

func (r *Runner) listen(ctx context.Context, flow *core.Flow, events <-chan core.TriggerEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			r.logger.Info("trigger event", "flow_id", event.FlowID, "time", event.Timestamp)
			if _, err := r.RunOnce(ctx, flow); err != nil {
				r.logger.Error("flow run failed", "error", err)
			}
		}
	}
}
