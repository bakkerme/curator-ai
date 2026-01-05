package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bakkerme/curator-ai/internal/core"
)

type Runner struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{logger: logger}
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

	blocks := []*core.PostBlock{}
	for _, source := range flow.Sources {
		if source == nil {
			continue
		}
		fetched, err := source.Fetch(ctx)
		if err != nil {
			run.Status = core.RunStatusFailed
			return run, err
		}
		blocks = append(blocks, fetched...)
	}

	for _, processor := range flow.Quality {
		if processor == nil {
			continue
		}
		next, err := processor.Evaluate(ctx, blocks)
		if err != nil {
			run.Status = core.RunStatusFailed
			return run, err
		}
		blocks = next
	}

	for _, processor := range flow.PostSummary {
		if processor == nil {
			continue
		}
		next, err := processor.Summarize(ctx, blocks)
		if err != nil {
			run.Status = core.RunStatusFailed
			return run, err
		}
		blocks = next
	}

	var runSummary *core.RunSummary
	for _, processor := range flow.RunSummary {
		if processor == nil {
			continue
		}
		summary, err := processor.SummarizeRun(ctx, blocks)
		if err != nil {
			run.Status = core.RunStatusFailed
			return run, err
		}
		runSummary = summary
	}

	for _, output := range flow.Outputs {
		if output == nil {
			continue
		}
		if err := output.Deliver(ctx, blocks, runSummary); err != nil {
			run.Status = core.RunStatusFailed
			return run, err
		}
	}

	completedAt := time.Now().UTC()
	run.CompletedAt = &completedAt
	run.Status = core.RunStatusCompleted
	run.Blocks = blocks
	run.RunSummary = runSummary
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
