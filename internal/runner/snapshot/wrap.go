package snapshot

import "github.com/bakkerme/curator-ai/internal/core"

type ConfigProvider interface {
	SnapshotConfig() *core.SnapshotConfig
}

type SourceWrapper struct {
	core.SourceProcessor
	snapshot *core.SnapshotConfig
}

func (w *SourceWrapper) SnapshotConfig() *core.SnapshotConfig {
	return w.snapshot
}

type QualityWrapper struct {
	core.QualityProcessor
	snapshot *core.SnapshotConfig
}

func (w *QualityWrapper) SnapshotConfig() *core.SnapshotConfig {
	return w.snapshot
}

type SummaryWrapper struct {
	core.SummaryProcessor
	snapshot *core.SnapshotConfig
}

func (w *SummaryWrapper) SnapshotConfig() *core.SnapshotConfig {
	return w.snapshot
}

type RunSummaryWrapper struct {
	core.RunSummaryProcessor
	snapshot *core.SnapshotConfig
}

func (w *RunSummaryWrapper) SnapshotConfig() *core.SnapshotConfig {
	return w.snapshot
}

type OutputWrapper struct {
	core.OutputProcessor
	snapshot *core.SnapshotConfig
}

func (w *OutputWrapper) SnapshotConfig() *core.SnapshotConfig {
	return w.snapshot
}

func WrapSource(processor core.SourceProcessor, cfg *core.SnapshotConfig) core.SourceProcessor {
	if processor == nil {
		return nil
	}
	if cfg == nil {
		return processor
	}
	return &SourceWrapper{SourceProcessor: processor, snapshot: cfg}
}

func WrapQuality(processor core.QualityProcessor, cfg *core.SnapshotConfig) core.QualityProcessor {
	if processor == nil {
		return nil
	}
	if cfg == nil {
		return processor
	}
	return &QualityWrapper{QualityProcessor: processor, snapshot: cfg}
}

func WrapSummary(processor core.SummaryProcessor, cfg *core.SnapshotConfig) core.SummaryProcessor {
	if processor == nil {
		return nil
	}
	if cfg == nil {
		return processor
	}
	return &SummaryWrapper{SummaryProcessor: processor, snapshot: cfg}
}

func WrapRunSummary(processor core.RunSummaryProcessor, cfg *core.SnapshotConfig) core.RunSummaryProcessor {
	if processor == nil {
		return nil
	}
	if cfg == nil {
		return processor
	}
	return &RunSummaryWrapper{RunSummaryProcessor: processor, snapshot: cfg}
}

func WrapOutput(processor core.OutputProcessor, cfg *core.SnapshotConfig) core.OutputProcessor {
	if processor == nil {
		return nil
	}
	if cfg == nil {
		return processor
	}
	return &OutputWrapper{OutputProcessor: processor, snapshot: cfg}
}
