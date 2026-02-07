package summary

import (
	"context"
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type echoClient struct{}

func (c *echoClient) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	_ = ctx
	if len(request.Messages) == 0 {
		return llm.ChatResponse{Content: ""}, nil
	}
	last := request.Messages[len(request.Messages)-1]
	return llm.ChatResponse{Content: last.Content}, nil
}

func TestPostLLMProcessor_Summarize_PerChunk(t *testing.T) {
	client := &echoClient{}
	cfg := &config.LLMSummary{
		Name:           "s",
		Type:           "summary_llm",
		Context:        "post",
		SystemTemplate: "system",
		PromptTemplate: "full={{.Content}}",
		ChunkSystem:    "chunk-system",
		ChunkPrompt:    "chunk={{.Chunk.Content}}",
	}
	processor, err := NewPostLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewPostLLMProcessor error: %v", err)
	}
	blocks := []*core.PostBlock{{
		ID:          "1",
		Content:     "full",
		Chunks:      []core.ContentChunk{{Content: "a"}, {Content: "b"}},
		SummaryPlan: &core.SummaryPlan{Mode: core.SummaryModePerChunk},
	}}
	updated, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if updated[0].Summary != nil {
		t.Fatalf("expected no final summary for per_chunk, got %#v", updated[0].Summary)
	}
	if got := updated[0].Chunks[0].Summary; got != "chunk=a" {
		t.Fatalf("expected chunk 0 summary 'chunk=a', got %q", got)
	}
	if got := updated[0].Chunks[1].Summary; got != "chunk=b" {
		t.Fatalf("expected chunk 1 summary 'chunk=b', got %q", got)
	}
}

func TestPostLLMProcessor_Summarize_MapReduce(t *testing.T) {
	client := &echoClient{}
	cfg := &config.LLMSummary{
		Name:           "s",
		Type:           "summary_llm",
		Context:        "post",
		SystemTemplate: "system",
		PromptTemplate: "final={{range .ChunkSummaries}}{{.}}|{{end}}",
		ChunkSystem:    "chunk-system",
		ChunkPrompt:    "chunk={{.Chunk.Content}}",
	}
	processor, err := NewPostLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewPostLLMProcessor error: %v", err)
	}
	blocks := []*core.PostBlock{{
		ID:          "1",
		Content:     "full",
		Chunks:      []core.ContentChunk{{Content: "a"}, {Content: "b"}},
		SummaryPlan: &core.SummaryPlan{Mode: core.SummaryModeMapReduce},
	}}
	updated, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if updated[0].Summary == nil {
		t.Fatalf("expected final summary to be set")
	}
	if !strings.Contains(updated[0].Summary.Summary, "chunk=a|") || !strings.Contains(updated[0].Summary.Summary, "chunk=b|") {
		t.Fatalf("expected final summary to include chunk summaries, got %q", updated[0].Summary.Summary)
	}
}

func TestPostLLMProcessor_Summarize_MissingSummaryPlanDefaultsToFull(t *testing.T) {
	client := &echoClient{}
	cfg := &config.LLMSummary{
		Name:           "s",
		Type:           "summary_llm",
		Context:        "post",
		SystemTemplate: "system",
		PromptTemplate: "full={{.Content}}",
		ChunkSystem:    "chunk-system",
		ChunkPrompt:    "chunk={{.Chunk.Content}}",
	}
	processor, err := NewPostLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewPostLLMProcessor error: %v", err)
	}
	blocks := []*core.PostBlock{{ID: "1", Content: "full"}}
	updated, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if updated[0].Summary == nil {
		t.Fatalf("expected final summary to be set")
	}
	if got := updated[0].Summary.Summary; got != "full=full" {
		t.Fatalf("expected full-mode summary, got %q", got)
	}
	if updated[0].SummaryPlan == nil || updated[0].SummaryPlan.Mode != core.SummaryModeFull {
		t.Fatalf("expected summary plan mode to default to full, got %#v", updated[0].SummaryPlan)
	}
}
