package summary

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

func TestPostLLMProcessor_Summarize_Parallel(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	client := &blockingClient{
		started: make(chan struct{}, 3),
		release: release,
		resp:    llm.ChatResponse{Content: "ok"},
	}
	cfg := &config.LLMSummary{
		Name:           "s",
		Type:           "summary_llm",
		Context:        "post",
		SystemTemplate: "system",
		PromptTemplate: "title={{.Title}}",
		MaxConcurrency: 3,
	}
	processor, err := NewPostLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewPostLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{ID: "1", Title: "a"}, {ID: "2", Title: "b"}, {ID: "3", Title: "c"}}
	done := make(chan error, 1)
	go func() {
		_, err := processor.Summarize(context.Background(), blocks)
		done <- err
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
	case err := <-done:
		if err != nil {
			t.Fatalf("Summarize error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Summarize to complete")
	}

	for i, block := range blocks {
		if block.Summary == nil || block.Summary.Summary == "" {
			t.Fatalf("expected summary for block %d, got %#v", i, block.Summary)
		}
	}
}

func TestPostLLMProcessor_Summarize_BlockErrorPolicyDrop_Parallel(t *testing.T) {
	t.Parallel()

	client := &conditionalClient{
		errWhen: "title=b",
		resp:    llm.ChatResponse{Content: "ok"},
	}
	cfg := &config.LLMSummary{
		Name:             "s",
		Type:             "summary_llm",
		Context:          "post",
		SystemTemplate:   "system",
		PromptTemplate:   "title={{.Title}}",
		MaxConcurrency:   3,
		BlockErrorPolicy: config.BlockErrorPolicyDrop,
	}
	processor, err := NewPostLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewPostLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{ID: "1", Title: "a"}, {ID: "2", Title: "b"}, {ID: "3", Title: "c"}}
	filtered, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(filtered))
	}
	if filtered[0].ID != "1" || filtered[1].ID != "3" {
		t.Fatalf("expected IDs [1 3], got [%s %s]", filtered[0].ID, filtered[1].ID)
	}
	if filtered[0].Summary == nil || filtered[0].Summary.Summary == "" {
		t.Fatalf("expected summary for block 1, got %#v", filtered[0].Summary)
	}
	if filtered[1].Summary == nil || filtered[1].Summary.Summary == "" {
		t.Fatalf("expected summary for block 3, got %#v", filtered[1].Summary)
	}
}

func TestPostLLMProcessor_Summarize_BlockErrorPolicyDrop_Serial(t *testing.T) {
	t.Parallel()

	client := &conditionalClient{
		errWhen: "title=b",
		resp:    llm.ChatResponse{Content: "ok"},
	}
	cfg := &config.LLMSummary{
		Name:             "s",
		Type:             "summary_llm",
		Context:          "post",
		SystemTemplate:   "system",
		PromptTemplate:   "title={{.Title}}",
		MaxConcurrency:   1,
		BlockErrorPolicy: config.BlockErrorPolicyDrop,
	}
	processor, err := NewPostLLMProcessor(cfg, client, "default-model")
	if err != nil {
		t.Fatalf("NewPostLLMProcessor error: %v", err)
	}

	blocks := []*core.PostBlock{{ID: "1", Title: "a"}, {ID: "2", Title: "b"}, {ID: "3", Title: "c"}}
	filtered, err := processor.Summarize(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(filtered))
	}
	if filtered[0].ID != "1" || filtered[1].ID != "3" {
		t.Fatalf("expected IDs [1 3], got [%s %s]", filtered[0].ID, filtered[1].ID)
	}
}
