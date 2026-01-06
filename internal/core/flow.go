package core

import (
	"time"
)

// TriggerProcessor represents a processor that includes only one processor type, alongside the Name and Type.
// It's done like this to avoid having to use a map[string]interface{} for the processor type.
type ProcessReference struct {
	Name        string
	Type        ProcessorType
	Trigger     TriggerProcessor
	Source      SourceProcessor
	Quality     QualityProcessor
	PostSummary SummaryProcessor
	RunSummary  RunSummaryProcessor
	Output      OutputProcessor
}

// Flow represents the internal structure of a parsed Curator Document
// It contains the configuration and all processors in their execution order
type Flow struct {
	ID                string                 `json:"id" yaml:"id"`
	Name              string                 `json:"name" yaml:"name"`
	Version           string                 `json:"version,omitempty" yaml:"version,omitempty"`
	CreatedAt         time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" yaml:"updated_at"`
	Status            FlowStatus             `json:"status" yaml:"status"`
	Triggers          []TriggerProcessor     `json:"-" yaml:"-"`
	Sources           []SourceProcessor      `json:"-" yaml:"-"`
	Quality           []QualityProcessor     `json:"-" yaml:"-"`
	PostSummary       []SummaryProcessor     `json:"-" yaml:"-"`
	RunSummary        []RunSummaryProcessor  `json:"-" yaml:"-"`
	Outputs           []OutputProcessor      `json:"-" yaml:"-"`
	RawConfig         map[string]interface{} `json:"raw_config" yaml:"raw_config"`
	OrderOfOperations []ProcessReference     `json:"-" yaml:"-"` // These point to the attached processors seen in in the flow above, in the resolved order of operations
}

// FlowStatus represents the current state of a flow
type FlowStatus string

const (
	FlowStatusWaiting   FlowStatus = "waiting"
	FlowStatusRunning   FlowStatus = "running"
	FlowStatusCompleted FlowStatus = "completed"
	FlowStatusFailed    FlowStatus = "failed"
	FlowStatusCancelled FlowStatus = "cancelled"
)

// ProcessorConfig represents the configuration for a single processor
type ProcessorConfig struct {
	Type   string                 `json:"type" yaml:"type"`
	Name   string                 `json:"name" yaml:"name"`
	Config map[string]interface{} `json:"config" yaml:"config,inline"`
}

// Run represents a single execution of a Flow
type Run struct {
	ID          string                 `json:"id" yaml:"id"`
	FlowID      string                 `json:"flow_id" yaml:"flow_id"`
	StartedAt   time.Time              `json:"started_at" yaml:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	Status      RunStatus              `json:"status" yaml:"status"`
	TriggerType string                 `json:"trigger_type" yaml:"trigger_type"`
	Blocks      []*PostBlock           `json:"blocks,omitempty" yaml:"blocks,omitempty"`
	RunSummary  *RunSummary            `json:"run_summary,omitempty" yaml:"run_summary,omitempty"`
	Errors      []ProcessError         `json:"errors,omitempty" yaml:"errors,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// RunStatus represents the current state of a run
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)
