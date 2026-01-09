package quality

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/llm"
)

type conditionalClient struct {
	mu      sync.Mutex
	errWhen string
	resp    llm.ChatResponse
}

func (c *conditionalClient) ChatCompletion(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range request.Messages {
		if strings.Contains(m.Content, c.errWhen) {
			return llm.ChatResponse{}, errors.New("boom")
		}
	}
	return c.resp, nil
}

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

func TestLLMProcessor_Evaluate_BlockErrorPolicyDrop_Parallel(t *testing.T) {
	t.Parallel()

	client := &conditionalClient{
		errWhen: "title=b",
		resp:    llm.ChatResponse{Content: `{"score":0.9,"reason":"ok"}`},
	}
	cfg := &config.LLMQuality{
		Name:             "q",
		Model:            "test-model",
		SystemTemplate:   "system",
		PromptTemplate:   "title={{.Title}}",
		Threshold:        0.5,
		MaxConcurrency:   3,
		BlockErrorPolicy: config.BlockErrorPolicyDrop,
	}
	processor, err := NewLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{ID: "1", Title: "a"}, {ID: "2", Title: "b"}, {ID: "3", Title: "c"}}
	filtered, err := processor.Evaluate(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered blocks, got %d", len(filtered))
	}
	if filtered[0].ID != "1" || filtered[1].ID != "3" {
		t.Fatalf("expected IDs [1 3], got [%s %s]", filtered[0].ID, filtered[1].ID)
	}
	if blocks[0].Quality == nil || blocks[0].Quality.Result != "pass" {
		t.Fatalf("expected pass for block 1, got %#v", blocks[0].Quality)
	}
	if blocks[2].Quality == nil || blocks[2].Quality.Result != "pass" {
		t.Fatalf("expected pass for block 3, got %#v", blocks[2].Quality)
	}
}

func TestLLMProcessor_Evaluate_BlockErrorPolicyDrop_Serial(t *testing.T) {
	t.Parallel()

	client := &conditionalClient{
		errWhen: "title=b",
		resp:    llm.ChatResponse{Content: `{"score":0.9,"reason":"ok"}`},
	}
	cfg := &config.LLMQuality{
		Name:             "q",
		Model:            "test-model",
		SystemTemplate:   "system",
		PromptTemplate:   "title={{.Title}}",
		Threshold:        0.5,
		MaxConcurrency:   1,
		BlockErrorPolicy: config.BlockErrorPolicyDrop,
	}
	processor, err := NewLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{ID: "1", Title: "a"}, {ID: "2", Title: "b"}, {ID: "3", Title: "c"}}
	filtered, err := processor.Evaluate(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered blocks, got %d", len(filtered))
	}
	if filtered[0].ID != "1" || filtered[1].ID != "3" {
		t.Fatalf("expected IDs [1 3], got [%s %s]", filtered[0].ID, filtered[1].ID)
	}
}
