package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bakkerme/curator-ai/internal/core"
)

type Payload struct {
	Blocks     []*core.PostBlock `json:"blocks"`
	RunSummary *core.RunSummary  `json:"run_summary,omitempty"`
}

func Save(path string, blocks []*core.PostBlock, runSummary *core.RunSummary) error {
	if path == "" {
		return fmt.Errorf("snapshot path is required")
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create snapshot directory: %w", err)
		}
	}
	payload := Payload{
		Blocks:     blocks,
		RunSummary: runSummary,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

func Load(path string) ([]*core.PostBlock, *core.RunSummary, error) {
	if path == "" {
		return nil, nil, fmt.Errorf("snapshot path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read snapshot: %w", err)
	}
	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return payload.Blocks, payload.RunSummary, nil
}
