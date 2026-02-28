package testfile

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

// TestFileProcessor loads a markdown file from disk and emits it as a post with optional chunking.
type TestFileProcessor struct {
	name   string
	config config.TestFileSource
}

func NewTestFileProcessor(cfg *config.TestFileSource) (*TestFileProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("testfile config is required")
	}
	return &TestFileProcessor{name: "testfile", config: *cfg}, nil
}

func (p *TestFileProcessor) Name() string {
	return p.name
}

func (p *TestFileProcessor) Configure(config map[string]interface{}) error {
	return nil
}

func (p *TestFileProcessor) Validate() error {
	if strings.TrimSpace(p.config.Path) == "" {
		return fmt.Errorf("testfile path is required")
	}
	return nil
}

func (p *TestFileProcessor) Fetch(ctx context.Context) ([]*core.PostBlock, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	_ = ctx
	data, err := os.ReadFile(p.config.Path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	chunks := chunkContent(content, p.config.ChunkSize)
	block := &core.PostBlock{
		ID:          "testfile",
		Title:       "Test File",
		Content:     content,
		CreatedAt:   time.Now().UTC(),
		ProcessedAt: time.Now().UTC(),
		SummaryPlan: summaryPlanFromConfig(p.config.SummaryPlan),
		Chunks:      chunks,
	}
	return []*core.PostBlock{block}, nil
}

func chunkContent(content string, chunkSize int) []core.ContentChunk {
	if chunkSize <= 0 {
		return []core.ContentChunk{{Content: content}}
	}
	runes := []rune(content)
	chunks := make([]core.ContentChunk, 0, (len(runes)+chunkSize-1)/chunkSize)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, core.ContentChunk{Content: string(runes[start:end])})
	}
	return chunks
}
