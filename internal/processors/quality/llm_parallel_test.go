package quality

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type blockingClient struct {
	started chan struct{}
	release <-chan struct{}
	current int32
	max     int32
	resp    llm.ChatResponse
}

func (c *blockingClient) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	_ = request
	n := atomic.AddInt32(&c.current, 1)
	for {
		old := atomic.LoadInt32(&c.max)
		if n <= old {
			break
		}
		if atomic.CompareAndSwapInt32(&c.max, old, n) {
			break
		}
	}
	c.started <- struct{}{}
	select {
	case <-c.release:
	case <-ctx.Done():
	}
	atomic.AddInt32(&c.current, -1)
	if err := ctx.Err(); err != nil {
		return llm.ChatResponse{}, err
	}
	return c.resp, nil
}

func TestLLMProcessor_Evaluate_Parallel(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	client := &blockingClient{
		started: make(chan struct{}, 3),
		release: release,
		resp:    llm.ChatResponse{Content: `{"score":0.9,"reason":"ok"}`},
	}
	cfg := &config.LLMQuality{
		Name:           "q",
		Model:          "test-model",
		SystemTemplate: "system",
		PromptTemplate: "title={{.Title}}",
		Threshold:      0.5,
		MaxConcurrency: 3,
	}
	processor, err := NewLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{ID: "1", Title: "a"}, {ID: "2", Title: "b"}, {ID: "3", Title: "c"}}
	done := make(chan struct{})
	var filtered []*core.PostBlock
	var evalErr error
	go func() {
		filtered, evalErr = processor.Evaluate(context.Background(), blocks)
		close(done)
	}()

	for i := 0; i < len(blocks); i++ {
		select {
		case <-client.started:
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for LLM calls to start (%d/%d)", i, len(blocks))
		}
	}
	if got := atomic.LoadInt32(&client.max); got != int32(len(blocks)) {
		t.Fatalf("expected max in-flight=%d, got %d", len(blocks), got)
	}

	close(release)
	select {
	case <-done:
		if evalErr != nil {
			t.Fatalf("Evaluate error: %v", evalErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Evaluate to complete")
	}

	if len(filtered) != len(blocks) {
		t.Fatalf("expected %d filtered blocks, got %d", len(blocks), len(filtered))
	}
	for i, block := range blocks {
		if block.Quality == nil || block.Quality.Result != "pass" {
			t.Fatalf("expected pass for block %d, got %#v", i, block.Quality)
		}
	}
}
