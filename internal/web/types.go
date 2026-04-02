package web

import (
	"time"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

func MapCoreRunStatus(s string) RunStatus {
	switch s {
	case "running":
		return RunStatusRunning
	case "completed":
		return RunStatusCompleted
	case "failed":
		return RunStatusFailed
	case "cancelled":
		return RunStatusCancelled
	default:
		return RunStatusPending
	}
}

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	RunID   string    `json:"run_id"`
	FlowID  string    `json:"flow_id"`
	Stage   string    `json:"stage,omitempty"`
}

type RunInfo struct {
	ID          string     `json:"id"`
	FlowID      string     `json:"flow_id"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      RunStatus  `json:"status"`
	Error       string     `json:"error,omitempty"`
}

type DocStatus struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SourcePath  string   `json:"source_path"`
	LastRun     *RunInfo `json:"last_run,omitempty"`
	IsRunning   bool     `json:"is_running"`
	ActiveRunID string   `json:"active_run_id,omitempty"`
}

type ListDocsResponse struct {
	Docs []DocStatus `json:"docs"`
}

type TriggerRunResponse struct {
	RunID     string    `json:"run_id"`
	FlowID    string    `json:"flow_id"`
	StartedAt time.Time `json:"started_at"`
}

type TriggerRunRequest struct {
}

type APIResponse struct {
	Error string `json:"error,omitempty"`
}
