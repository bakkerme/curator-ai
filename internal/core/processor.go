package core

import (
	"context"
	"time"
)

type ProcessorType string

var TriggerProcessorType ProcessorType = "trigger_processor"
var SourceProcessorType ProcessorType = "source_processor"
var QualityProcessorType ProcessorType = "quality_processor"
var SummaryProcessorType ProcessorType = "summary_processor"
var RunSummaryProcessorType ProcessorType = "run_summary_processor"
var OutputProcessorType ProcessorType = "output_processor"

// Processor is the base interface that all processors must implement
type Processor interface {
	// Name returns the processor name
	Name() string
	// Configure sets up the processor with the provided configuration
	Configure(config map[string]interface{}) error
	// Validate checks if the processor configuration is valid
	Validate() error
}

type SnapshotConfig struct {
	Snapshot bool   `json:"snapshot" yaml:"snapshot"`
	Restore  bool   `json:"restore" yaml:"restore"`
	Path     string `json:"path" yaml:"path"`
}

// TriggerEvent represents a trigger firing
type TriggerEvent struct {
	FlowID    string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// TriggerProcessor defines when processing runs
type TriggerProcessor interface {
	Processor
	// Start begins the trigger and returns a channel of trigger events
	// The processor manages its own lifecycle and sends events when triggered.
	// Each processor is specifically instantiated with its relevant configuration for the flow.
	Start(ctx context.Context, flowID string) (<-chan TriggerEvent, error)
	// Stop gracefully shuts down the trigger
	Stop() error
}

// SourceProcessor fetches and ingests data, creating blocks
type SourceProcessor interface {
	Processor
	// Fetch retrieves data from the source and creates blocks
	Fetch(ctx context.Context) ([]*PostBlock, error)
}

// QualityProcessor filters and evaluates content
type QualityProcessor interface {
	Processor
	// Evaluate processes blocks and determines quality
	// Returns the same blocks with Quality field populated
	Evaluate(ctx context.Context, blocks []*PostBlock) ([]*PostBlock, error)
}

// SummaryProcessor transforms and summarizes content
type SummaryProcessor interface {
	Processor
	// Summarize processes blocks and creates summaries
	// Returns the same blocks with Summary field populated
	Summarize(ctx context.Context, blocks []*PostBlock) ([]*PostBlock, error)
}

// RunSummaryProcessor creates aggregate summaries across all posts
type RunSummaryProcessor interface {
	Processor
	// SummarizeRun creates a summary across all blocks in a run
	SummarizeRun(ctx context.Context, blocks []*PostBlock, current *RunSummary) (*RunSummary, error)
}

// OutputProcessor delivers results
type OutputProcessor interface {
	Processor
	// Deliver sends the processed blocks to the configured output
	Deliver(ctx context.Context, blocks []*PostBlock, runSummary *RunSummary) error
}
