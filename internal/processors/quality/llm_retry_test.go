package quality

import (
	"context"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
	llmmock "github.com/bakkerme/curator-ai/internal/llm/mock"
)

func TestLLMProcessor_RetriesOnInvalidJSON(t *testing.T) {
	t.Parallel()

	cfg := &config.LLMQuality{
		Name:               "q",
		Model:              "test-model",
		SystemTemplate:     "system",
		PromptTemplate:     "title={{.Title}}",
		Threshold:          0.5,
		InvalidJSONRetries: 1,
	}
	client := &llmmock.Client{
		Responses: []llm.ChatResponse{
			{Content: "not json"},
			{Content: `{"score":0.9,"reason":"ok"}`},
		},
	}
	processor, err := NewLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{Title: "hello"}}
	filtered, err := processor.Evaluate(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered block, got %d", len(filtered))
	}
	if got := len(client.Calls); got != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", got)
	}
	if blocks[0].Quality == nil || blocks[0].Quality.Result != "pass" {
		t.Fatalf("expected block to pass quality, got: %#v", blocks[0].Quality)
	}
}
